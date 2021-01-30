package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/urfave/cli"
)

var (
	secrets     = false
	profile     = ""
	noNewLine   = false
	hideTS      = false
	stripPrefix = false
)

func main() {
	app := cli.NewApp()
	app.Version = "1.3.4"

	app.Usage = "simple ssm param store interface"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "profile, p",
			Usage:       "Specify an AWS profile. Optional. Defaults to AWS_PROFILE.",
			Destination: &profile,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "ls",
			Usage: "list param names. ex: ssm ls myapp, ssm ls, ssm ls --secrets myapp",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "secrets",
					Usage:       "print out parameter values in plaintext",
					Destination: &secrets,
				},
				cli.BoolFlag{
					Name:        "hide-ts",
					Usage:       "prints keys in alphabetical order without timestamps (good for diffs)",
					Destination: &hideTS,
				},
				cli.BoolFlag{
					Name:        "strip-prefix",
					Usage:       "strips prefix from the variable (also good for diffs)",
					Destination: &stripPrefix,
				},
			},
			Action: func(c *cli.Context) error {
				if profile != "" {
					err := os.Setenv("AWS_PROFILE", profile)
					if err != nil {
						return err
					}
				}
				log.Println("fetching ssm keys")
				s := c.Args().First()
				keys, err := list(s, secrets, !hideTS, stripPrefix)
				if err != nil {
					return err
				}

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
				if secrets {
					fmt.Fprintln(w, "Last Modified\tKey\tValue")
				} else {
					fmt.Fprintln(w, "Last Modified\tKey")
				}
				for _, k := range keys {
					fmt.Fprintln(w, k)
				}
				err = w.Flush()
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:  "get",
			Usage: "prints plaintext ssm value. ex: ssm get /app/prod/my-key",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:        "n",
					Usage:       "Do not print a trailing newline",
					Destination: &noNewLine,
				},
			},
			Action: func(c *cli.Context) error {
				if profile != "" {
					err := os.Setenv("AWS_PROFILE", profile)
					if err != nil {
						return err
					}
				}
				key := c.Args().First()
				val, err := get(key)
				if err != nil {
					return err
				}
				if noNewLine {
					fmt.Print(val)
				} else {
					fmt.Println(val)
				}
				return nil
			},
		},

		{
			Name:  "set",
			Usage: "sets ssm k,v pair. overwrites. ex: ssm set /app/prod/version 27",
			Action: func(c *cli.Context) error {
				if profile != "" {
					err := os.Setenv("AWS_PROFILE", profile)
					if err != nil {
						return err
					}
				}
				key := c.Args().First()
				val := c.Args().Get(1)
				err := set(key, val)
				return err
			},
		},

		{
			Name:  "rm",
			Usage: "removes ssm param. ex: ssm rm /app/prod/param",
			Action: func(c *cli.Context) error {
				if profile != "" {
					err := os.Setenv("AWS_PROFILE", profile)
					if err != nil {
						return err
					}
				}
				key := c.Args().First()
				err := rm(key)
				return err
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func rm(key string) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ssmsvc := ssm.New(sess, aws.NewConfig())
	_, err := ssmsvc.DeleteParameter(&ssm.DeleteParameterInput{
		Name: &key,
	})
	return err
}

func set(key, val string) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ssmsvc := ssm.New(sess, aws.NewConfig())
	overwrite := true
	ptype := "SecureString"
	tier := "Standard"
	if len([]byte(val)) > 4096 {
		tier = "Advanced"
	}
	_, err := ssmsvc.PutParameter(&ssm.PutParameterInput{
		Name:      &key,
		Value:     &val,
		Overwrite: &overwrite,
		Type:      &ptype,
		Tier:      &tier,
	})
	return err
}

type entry struct {
	t    *time.Time
	name string
	val  string
}

func (e *entry) fmt(ts, stripPrefix bool) string {
	var val string
	if stripPrefix {
		s := strings.Split(e.val, "/")
		val = s[len(s)-1]
	} else {
		val = e.val
	}
	if ts {
		return strings.Join([]string{e.t.Format("2006-01-02 15:04:05"), e.name, val}, "\t")
	}
	return strings.Join([]string{e.name, val}, "\t")
}

func list(s string, showValue, ts, stripPrefix bool) ([]string, error) {
	// build aws session
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	ssmsvc := ssm.New(sess, aws.NewConfig())
	var next string
	var n int64 = 50

	// set name filters for AWS
	k := "Name"
	filterOption := "Contains"
	filter := ssm.ParameterStringFilter{Key: &k, Option: &filterOption, Values: []*string{&s}}
	var in ssm.DescribeParametersInput

	// if filter specified, add name filters
	if s != "" {
		in = ssm.DescribeParametersInput{
			ParameterFilters: []*ssm.ParameterStringFilter{&filter},
		}
	} else {
		in = ssm.DescribeParametersInput{}
	}

	// blocking semaphore channel to keep concurrency under control
	semChan := make(chan struct{}, 5)
	defer close(semChan)

	params := []entry{}
	// iterate over results
	for {
		desc, err := ssmsvc.DescribeParameters(&in)
		if err != nil {
			return []string{}, err
		}
		// result channel to store entries from concurrent secret requests
		resultChan := make(chan entry, len(desc.Parameters))
		defer close(resultChan)
		for _, p := range desc.Parameters {
			if p.Name != nil {
				name := *p.Name
				date := p.LastModifiedDate
				if showValue {
					semChan <- struct{}{}

					go func() {
						v, err := get(name)
						if err != nil {
							log.Fatal(err)
						} else {
							resultChan <- entry{date, name, v}
						}
						<-semChan
					}()
				} else {
					resultChan <- entry{date, name, ""}
				}
			}
		}
		for i := 0; i < len(desc.Parameters); i++ {
			p := <-resultChan
			params = append(params, p)
		}

		if desc.NextToken != nil {
			next = *desc.NextToken
			if s != "" {
				in = ssm.DescribeParametersInput{NextToken: &next, MaxResults: &n, ParameterFilters: []*ssm.ParameterStringFilter{&filter}}
			} else {
				in = ssm.DescribeParametersInput{NextToken: &next, MaxResults: &n}
			}
		} else {
			break
		}
	}

	if ts {
		sort.Slice(params, func(i, j int) bool {
			return params[i].t.Before(*params[j].t)
		})

	} else {
		sort.Slice(params, func(i, j int) bool {
			return params[i].name < params[j].name
		})
	}

	vals := make([]string, 0)
	for _, p := range params {
		vals = append(vals, p.fmt(ts, stripPrefix))
	}

	return vals, nil
}

func get(key string) (string, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ssmsvc := ssm.New(sess, aws.NewConfig())
	withDecryption := true
	param, err := ssmsvc.GetParameter(&ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return "", err
	}

	value := *param.Parameter.Value
	return value, nil
}
