package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Version = "0.1.0"
	app.Usage = "simple ssm param store interface"
	app.Commands = []cli.Command{
		{
			Name:  "ls",
			Usage: "list param names. ex: ssm ls myapp, ssm ls",
			Action: func(c *cli.Context) error {
				log.Println("fetching ssm keys")
				s := c.Args().First()
				keys, err := list(s)
				if err != nil {
					return err
				}
				prettyPrint(keys)
				return nil
			},
		},
		{
			Name:  "get",
			Usage: "prints plaintext ssm value. ex: ssm get /app/prod/my-key",
			Action: func(c *cli.Context) error {
				key := c.Args().First()
				val, err := get(key)
				if err != nil {
					return err
				}
				fmt.Println(val)
				return nil
			},
		},

		{
			Name:  "set",
			Usage: "sets ssm k,v pair. overwrites. ex: ssm set /app/prod/version 27",
			Action: func(c *cli.Context) error {
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
	_, err := ssmsvc.PutParameter(&ssm.PutParameterInput{
		Name:      &key,
		Value:     &val,
		Overwrite: &overwrite,
		Type:      &ptype,
	})
	return err
}

func list(s string) ([]string, error) {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	ssmsvc := ssm.New(sess, aws.NewConfig())
	params := make([]string, 0)
	var next string
	var n int64 = 50
	var in ssm.DescribeParametersInput
	for {

		desc, err := ssmsvc.DescribeParameters(&in)
		if err != nil {
			return []string{}, err
		}
		for _, p := range desc.Parameters {
			if p.Name != nil {
				if s == "" || strings.Contains(strings.ToUpper(*p.Name), strings.ToUpper(s)) {
					params = append(params, *p.Name)
				}
			}
		}
		if desc.NextToken != nil {
			next = *desc.NextToken
			in = ssm.DescribeParametersInput{NextToken: &next, MaxResults: &n}
		} else {
			break
		}
	}

	return params, nil

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

// pretty print prints tab delimited keys based on the
// based on the width of the calling terminal
func prettyPrint(keys []string) {
	var batchSize int

	// get width of stdin -- if we can't figure it out,
	// guess 3 as the number of columns to pretty print
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		log.Println("could not get width of tty")
		batchSize = 3
	}

	// parse out the digits, the second digit is th width
	re := regexp.MustCompile("[0-9]+")

	matches := re.FindAll(out, 2)
	var width int
	if len(matches) == 2 {
		width, err = strconv.Atoi(string(matches[1]))
		if err != nil {
			batchSize = 3
		}
	} else {
		log.Println("could not get width of tty")
		batchSize = 3
	}

	// max is the largest keyname to print
	max := 0
	for _, k := range keys {
		if len(k) > max {
			max = len(k)
		}
	}

	// figure out how many columns you could have with the widest key
	if max > 0 {
		batchSize = (width / max)

	}

	// break up the keys into batches and print them
	// with a tabwriter
	var batches [][]string

	for batchSize < len(keys) {
		keys, batches = keys[batchSize:], append(batches, keys[0:batchSize:batchSize])
	}
	batches = append(batches, keys)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, batch := range batches {
		fmt.Fprintln(w, strings.Join(batch, "\t"))
	}
	w.Flush()
}
