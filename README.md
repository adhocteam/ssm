## SSM
### About
This is a command line interface to [AWS SSM Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/what-is-systems-manager.html).

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
