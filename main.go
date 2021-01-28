package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/urfave/cli"
)

var (
	Secrets   = false
	Profile   = ""
	NoNewLine = false
)

func main() {
	app := cli.NewApp()
	app.Version = "1.3.2"
	app.Usage = "simple ssm param store interface"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "profile, p",
			Usage:       "Specify an AWS profile. Optional. Defaults to AWS_PROFILE.",
			Destination: &Profile,
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
					Destination: &Secrets,
				},
			},
			Action: func(c *cli.Context) error {
				if Profile != "" {
					os.Setenv("AWS_PROFILE", Profile)
				}
				log.Println("fetching ssm keys")
				s := c.Args().First()
				keys, err := list(s, Secrets)
				if err != nil {
					return err
				}

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
				if Secrets {
					fmt.Fprintln(w, "Last Modified\tKey\tValue")
				} else {
					fmt.Fprintln(w, "Last Modified\tKey")
				}
				for _, k := range keys {
					fmt.Fprintln(w, k)
				}
				w.Flush()
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
					Destination: &NoNewLine,
				},
			},
			Action: func(c *cli.Context) error {
				if Profile != "" {
					os.Setenv("AWS_PROFILE", Profile)
				}
				key := c.Args().First()
				val, err := get(key)
				if err != nil {
					return err
				}
				if NoNewLine {
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
				if Profile != "" {
					os.Setenv("AWS_PROFILE", Profile)
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
				if Profile != "" {
					os.Setenv("AWS_PROFILE", Profile)
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

func (e *entry) fmt() string {
	return strings.Join([]string{e.t.Format("2006-01-02 15:04:05"), e.name, e.val}, "\t")
}

func list(s string, showValue bool) ([]string, error) {
	// build aws session
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	ssmsvc := ssm.New(sess, aws.NewConfig())
	params := make([]entry, 0)
	var next string
	var n int64 = 50

	// set name filters for AWS
	k := "Name"
	filterOption := "Contains"
	filter := ssm.ParameterStringFilter{Key: &k, Option: &filterOption, Values: []*string{&s}}
	var in ssm.DescribeParametersInput
	var wg sync.WaitGroup

	// if filter specified, add name filters
	if s != "" {
		in = ssm.DescribeParametersInput{
			ParameterFilters: []*ssm.ParameterStringFilter{&filter},
		}
	} else {
		in = ssm.DescribeParametersInput{}
	}
	// iterate over results

	// blocking semaphore channel to keep concurrency under control
	sem := make(chan struct{}, 5)
	for {
		desc, err := ssmsvc.DescribeParameters(&in)
		if err != nil {
			return []string{}, err
		}
		for _, p := range desc.Parameters {
			if p.Name != nil {
				name := *p.Name
				if showValue {
					// set waitgroup and fetch in a goroutine
					wg.Add(1)
					sem <- struct{}{}

					go func() {
						defer wg.Done()
						v, err := get(name)
						if err != nil {
							log.Fatal(err)
						} else {
							params = append(params,
								entry{p.LastModifiedDate, name, v},
							)

						}
						<-sem

					}()
				} else {
					params = append(params,
						entry{p.LastModifiedDate, *p.Name, ""},
					)
				}
			}
		}
		// let all goroutines finish
		wg.Wait()

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
	sort.Slice(params, func(i, j int) bool {
		return params[i].t.Before(*params[j].t)
	})

	vals := make([]string, 0)
	for _, p := range params {
		vals = append(vals, p.fmt())
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
