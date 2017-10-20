# awstagger
Tag AWS resources for beanstalk environment. Beanstalk doesn't tag everything to do with your AWS account. Automatically tag:
 - S3
 - Elasticache
 - Cloudwatch log stream
 - EBS volume
      
## How to use

You need to setup authentication ID and token in ~/.aws/credentials and ~/.aws./config for region settings. Fastest was is with AWS configure http://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html

```
cat ./config.json | ACCOUNT_ID=1111111111111 CLUSTER_NODES="node-001 node-002" REGION="ap-southeast-2" VERSION="0.1dev" go run ./main.go
```

## AWS golang Integration
https://docs.aws.amazon.com/sdk-for-go/api/
