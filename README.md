## SSM
### About

This is a command line interface to the [AWS SSM Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/what-is-systems-manager.html).

It's designed to be simple and easy to remember. A more featureful alternative is [Chamber](https://github.com/segmentio/chamber).

### Install

[Binaries](https://github.com/adhocteam/ssm/releases) are available for Linux and MacOS.

To install from source, install a [Go](https://golang.org/dl/) compiler and:

```
go install github.com/adhocteam/ssm@latest
```

### Usage

#### List params

All parameters:

```
ssm ls
```

Params containing `/my-app`

```
ssm ls /my-app
```

Params containing `/my-app` with secrets printed in plaintext

```
ssm ls --secrets /my-app
```

#### Get the value of a param

```
ssm get /myapp/staging/key
```

You can use the value of a parameter in a bash script like

```
PGPASSWORD=$(ssm get /myapp/prod/pgpass)
```

#### Set param key value

```
ssm set /myapp/staging/version 27
```

#### Delete param

```
ssm rm /myapp/staging/version
```

### Specifying the AWS Profile

The app will either rely on the `AWS_PROFILE` environment variable,
or you can set one with the `-p` and `--profile` flag. For each example above,
add that flag, i.e. `ssm -p myprofile ls myapp` to override the `AWS_PROFILE`.
