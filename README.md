![](https://codebuild.us-east-1.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoiNFl6UXNBbmNva1RUVFVTTXFQZk8wSkJoaTB2TnVtTkVvdXBVRi9QMXFmVWlsRzdZc0JNc1Z6Mi9qb3ZFeTFZbDR0YmRCM2V5enhZLzJDZFY5RGFaWXlZPSIsIml2UGFyYW1ldGVyU3BlYyI6ImRoR3NCbk0rMGRxeWRzTGUiLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=master)

## SSM
### About
This is a command line interface to [AWS SSM Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/what-is-systems-manager.html).

### Install
```
$ GO111MODULE=on go get -u github.com/adhocteam/ssm
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
