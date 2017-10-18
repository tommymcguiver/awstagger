package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/s3"
)

//Environement variables
var (
	accountID    = ""
	clusterNodes = ""
	region       = ""
	version      = ""
)

//JSON configuration file
var (
	Config map[string]string
)

func startup() {
	accountID = os.Getenv("ACCOUNT_ID")
	clusterNodes = os.Getenv("CLUSTER_NODES")
	//TODO: Get this from the AWS config
	region = os.Getenv("REGION")
	version = os.Getenv("VERSION")

	if accountID == "" || clusterNodes == "" || region == "" || version == "" {
		panic("Unable to continue please set environment variables")
	}
}

func readConfigFile() error {
	dec := json.NewDecoder(os.Stdin)
	//FIXME: Very naugty and nasty globals
	return dec.Decode(&Config)
}

//TODO: Replace version
//TODO: Add cache role
//TODO: Add verbosity
//TODO: Add help

func setup() *session.Session {
	// All clients require a Session. The Session provides the client with
	// shared configuration such as region, endpoint, and credentials. A
	// Session should be shared where possible to take advantage of
	// configuration and credential caching. See the session package for
	// more information.
	return session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
}

func tagS3() error {
	var s3Bucket = fmt.Sprintf("elasticbeanstalk-%s-%s", region, accountID)

	// Create a new instance of the service's client with a Session.
	// Optional aws.Config values can also be provided as variadic arguments
	// to the New function. This option allows you to provide service
	// specific configuration.
	svc := s3.New(setup())

	//Copy the tags into the coarrect format
	var tags []*s3.Tag
	for k, v := range Config {
		tags = append(tags, &s3.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	// fmt.Printf("%v\n", tags)

	input := &s3.PutBucketTaggingInput{
		Bucket: aws.String(s3Bucket),
		Tagging: &s3.Tagging{
			TagSet: tags,
		},
	}

	//Tag the bucket
	result, err := svc.PutBucketTagging(input)
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func tagElasticache() error {
	svc := elasticache.New(setup())
	//Copy the tags into the correct format
	var tags []*elasticache.Tag
	for k, v := range Config {
		tags = append(tags, &elasticache.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}

	for _, clusterID := range strings.Fields(clusterNodes) {
		input := &elasticache.AddTagsToResourceInput{
			ResourceName: aws.String(fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", region, accountID, clusterID)),
			Tags:         tags,
		}
		result, err := svc.AddTagsToResource(input)
		if err != nil {
			return err
		}

		fmt.Println(result)
	}

	return nil
}

func tagLogGroup() error {
	svc := cloudwatchlogs.New(setup())

	logs, _ := svc.DescribeLogGroups(&cloudwatchlogs.DescribeLogGroupsInput{})
	matcher := regexp.MustCompile("elasticbeanstalk")

	for _, logName := range logs.LogGroups {
		fmt.Printf("Found log group %s\n", *logName.LogGroupName)
		if matcher.MatchString(*logName.LogGroupName) {
			fmt.Printf("Tagging log group %s\n", *logName.LogGroupName)
			_, err := svc.TagLogGroup(&cloudwatchlogs.TagLogGroupInput{
				LogGroupName: logName.LogGroupName,
				Tags:         aws.StringMap(Config),
			})

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func tagEC2Volumes() error {

	svc := ec2.New(setup())

	//Find beanstalk instances
	instances, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("tag-key"),
				Values: []*string{aws.String("elasticbeanstalk:environment-id")},
			},
		},
	})

	if err != nil {
		return err
	}

	//Copy tags from EB instance to EBS volumes. Thanks AWS
	for _, r := range instances.Reservations {
		for _, i := range r.Instances {
			for _, b := range i.BlockDeviceMappings {
				fmt.Printf("Copying tags from ec2 %s to volume %s\n", *i.InstanceId, *b.Ebs.VolumeId)

				//Deep copy the tags
				var c []*ec2.Tag
				c = make([]*ec2.Tag, len(i.Tags))

				for index, t := range i.Tags {
					//Can't copy AWS keys, will get 4xx error
					if aws.StringValue(t.Key)[0:4] == "aws:" {
						continue
					}
					c[index] = &ec2.Tag{Key: aws.String(*t.Key), Value: aws.String(*t.Value)}
				}

				_, err := svc.CreateTags(&ec2.CreateTagsInput{
					Resources: []*string{aws.String(*b.Ebs.VolumeId)},
					Tags:      c,
				})

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

type promFunc func() error

//Create a promise style function
func promise(funcs []promFunc) {
	for _, f := range funcs {
		if err := f(); err != nil {
			panic(err)
		}
	}
}

/* Tag the AWS resources */
func main() {

	startup()

	promise([]promFunc{
		func() error {
			fmt.Println("Read config file")
			return readConfigFile()
		},
		func() error {
			fmt.Println("Tagging S3")
			return tagS3()
		},
		func() error {
			fmt.Println("Tagging Elasticache")
			return tagElasticache()
		},
		func() error {
			fmt.Println("Tagging Log Group")
			return tagLogGroup()
		},
		func() error {
			fmt.Println("Tagging EC2 Volumes")
			return tagEC2Volumes()
		},
	})
}
