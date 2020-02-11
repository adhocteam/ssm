![](https://codebuild.us-east-1.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoiNFl6UXNBbmNva1RUVFVTTXFQZk8wSkJoaTB2TnVtTkVvdXBVRi9QMXFmVWlsRzdZc0JNc1Z6Mi9qb3ZFeTFZbDR0YmRCM2V5enhZLzJDZFY5RGFaWXlZPSIsIml2UGFyYW1ldGVyU3BlYyI6ImRoR3NCbk0rMGRxeWRzTGUiLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=master)

## SSM
### About
This is a command line interface to [AWS SSM Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/what-is-systems-manager.html).

### Install
```
$ go get -u github.com/adhocteam/ssm
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

#### Get the value of a param
```
ssm get /myapp/staging/key
```

#### Set param key value
```
ssm set /myapp/staging/version 27
```

#### Delete param
```
ssm rm /myapp/staging/version
```
