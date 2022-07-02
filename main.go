package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/urfave/cli"
)

func main() {

	// show secrets in plaintext
	secrets := false
	// aws profile
	profile := ""
	// don't print newline on ssm get
	noNewLine := false
	// don't print timestamps of params
	hideTS := false
	// remove leading environment; /prod/version -> version
	stripPrefix := false
	// print tuple of parameter histories
	showHistory := false
	// serialize output to csv
	toCSV := false

	app := cli.NewApp()
	app.Version = "1.5.0"
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
					Name:        "csv",
					Usage:       "serialize output to csv",
					Destination: &toCSV,
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
				cli.BoolFlag{
					Name:        "show-history",
					Usage:       "if secrets are printed, print all previous versions with it ",
					Destination: &showHistory,
				},
			},
			Action: func(c *cli.Context) error {

				// create SSM session
				sess := session.Must(session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
					Profile:           profile,
				}))

				service := ssm.New(sess, aws.NewConfig())

				log.Println("fetching ssm keys")
				s := c.Args().First()

				// retrieve parameters
				keys, err := list(s, secrets, !hideTS, stripPrefix, showHistory, service)
				if err != nil {
					return err
				}

				if toCSV {
					w := csv.NewWriter(os.Stdout)
					w.Write([]string{"date", "key", "value"})
					for _, k := range keys {
						w.Write(k)
					}
					return err

				}
				for _, key := range keys {
					fmt.Println(strings.Join(key, "\t"))
				}

				return err
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
				// create SSM session
				sess := session.Must(session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
					Profile:           profile,
				}))

				service := ssm.New(sess, aws.NewConfig())
				key := c.Args().First()

				// fetch key
				val, err := get(key, service)
				if err != nil {
					return err
				}

				// print (with or without newline)
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
				// create SSM session
				sess := session.Must(session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
					Profile:           profile,
				}))

				service := ssm.New(sess, aws.NewConfig())

				// set key value pair
				key := c.Args().First()
				val := c.Args().Get(1)
				err := set(key, val, service)
				return err
			},
		},

		{
			Name:  "rm",
			Usage: "removes ssm param. ex: ssm rm /app/prod/param",
			Action: func(c *cli.Context) error {

				// create SSM session
				sess := session.Must(session.NewSessionWithOptions(session.Options{
					SharedConfigState: session.SharedConfigEnable,
					Profile:           profile,
				}))

				service := ssm.New(sess, aws.NewConfig())
				key := c.Args().First()
				// delete key
				err := rm(key, service)
				return err
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

// rm deletes a ssm key.
func rm(key string, service *ssm.SSM) error {

	_, err := service.DeleteParameter(&ssm.DeleteParameterInput{
		Name: &key,
	})
	return err
}

// set sets a ssm key to a value.
func set(key, val string, service *ssm.SSM) error {

	overwrite := true
	ptype := "SecureString"
	tier := "Standard"
	if len([]byte(val)) > 4096 {
		tier = "Advanced"
	}
	_, err := service.PutParameter(&ssm.PutParameterInput{
		Name:      &key,
		Value:     &val,
		Overwrite: &overwrite,
		Type:      &ptype,
		Tier:      &tier,
	})
	return err
}

// entry is a parameter entry used to format histories.
type entry struct {
	t       *time.Time
	name    string
	val     string
	history []string
}

func removePrefix(path string) string {
	s := strings.Split(path, "/")
	return s[len(s)-1]
}

// fmt returns a formatted string with optional timestamp and parameter prefix.
func (e *entry) fmt(ts, stripPrefix bool) []string {
	var val string
	if stripPrefix {
		val = removePrefix(e.val)
	} else {
		val = e.val
	}
	h := strings.Join(e.history, ", ")
	if ts {
		return []string{e.t.Format("2006-01-02 15:04:05"), e.name, val, h}
	}
	return []string{e.name, val, h}
}

// history returns the parameter history of a value.
func history(key string, service *ssm.SSM) ([]string, error) {

	hist := []string{}
	max := int64(50)
	decrypt := true
	in := ssm.GetParameterHistoryInput{MaxResults: &max, Name: &key, WithDecryption: &decrypt}
	for {
		out, err := service.GetParameterHistory(&in)
		if err != nil {
			return []string{}, err
		}
		for _, v := range out.Parameters {
			date := *v.LastModifiedDate
			hist = append(hist, fmt.Sprintf("(%s, %s)", date.Format("2006-01-02"), *v.Value))
		}
		if out.NextToken == nil {
			break
		}
		in = ssm.GetParameterHistoryInput{MaxResults: &max, NextToken: out.NextToken, Name: &key, WithDecryption: &decrypt}

	}
	return hist, nil
}

// list lists a set of parameters matching the substring s.
func list(s string, showValue, ts, stripPrefix, showHistory bool, service *ssm.SSM) ([][]string, error) {

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

	// set n workers based on how many requests may happen
	nworkers := 2

	if showHistory {
		nworkers = 1
	}
	// blocking semaphore channel to keep concurrency under control
	semChan := make(chan struct{}, nworkers)
	defer close(semChan)

	params := []entry{}
	// iterate over results
	for {
		desc, err := service.DescribeParameters(&in)
		if err != nil {
			return [][]string{}, err
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
						v, err := get(name, service)
						if err != nil {
							log.Fatal(err)
						}
						hist := []string{}
						if showHistory {
							hist, err = history(name, service)
							if err != nil {
								log.Fatal(err)
							}
						}
						resultChan <- entry{date, name, v, hist}

						<-semChan
					}()
				} else {
					resultChan <- entry{date, name, "", []string{}}
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

	vals := make([][]string, 0)
	for _, p := range params {
		vals = append(vals, p.fmt(ts, stripPrefix))
	}

	return vals, nil
}

// get gets the value of a parameter.
func get(key string, service *ssm.SSM) (string, error) {
	withDecryption := true
	param, err := service.GetParameter(&ssm.GetParameterInput{
		Name:           &key,
		WithDecryption: &withDecryption,
	})
	if err != nil {
		return "", err
	}

	value := *param.Parameter.Value
	return value, nil
}
