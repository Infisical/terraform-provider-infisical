package pkg

type AwsAuthenticationMethod string

const (
	AwsAuthMethodAccessKey  AwsAuthenticationMethod = "access_key"
	AwsAuthMethodAssumeRole AwsAuthenticationMethod = "assume_role"
)

type AwsAccessKeyCredentials struct {
	AccessKeyId     string
	SecretAccessKey string
}

type AwsAssumeRoleCredentials struct {
	AssumeRoleArn string
}
