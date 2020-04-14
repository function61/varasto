Create S3 bucket in AWS
-----------------------

Sign up for [AWS S3](https://aws.amazon.com/s3/).

Create a bucket, I named mine `myorg-varasto-fry` - why?

- `myorg` prefix because S3 bucket names have to be unique globally.

- `fry` (my Volume name) as suffix for clarity if I ever want to have multiple Varasto S3
  buckets with different S3 redundancy levels or other options/features.

Choose the region carefully (can affect pricing, latency etc.), as you can't easily change
the region for S3 bucket.

You can create the bucket with pretty much the default options.


Create access credentials
-------------------------

Move over to [IAM](https://console.aws.amazon.com/iam/home) to create highly restricted access
key for Varasto to only be able to access the bucket (not your other AWS resources).

Create new user - name doesn't matter but `varasto` would be good. Access type = `programmatic access`.

Don't enter any permissions or user groups. After creating the user, write down the
`access key id` and `secret access key` - you'll need these later in this guide.

Attach inline policy in JSON (replace your bucket name!):

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:PutObject",
                "s3:GetObject",
                "s3:PutObjectAcl"
            ],
            "Resource": "arn:aws:s3:::myorg-varasto-fry/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListBucket"
            ],
            "Resource": "arn:aws:s3:::myorg-varasto-fry*"
        }
    ]
}
```

You can use policy name: `varastoReadWriteBucket`


Create volume in Varasto
------------------------

Since this volume is stored in S3, the quota is effectively unlimited. But storage costs
money, so you should define the quota as the pain limit of what you're willing to pay AWS
for storage so you won't accidentally go over it.

Don't worry too much about defining the quota, since you can easily change it.


Mount the S3 bucket as volume in Varasto
----------------------------------------

Now mount the volume in Varasto. Options:

- `Bucket` = `myorg-varasto-fry` (replace this with the one you chose!)

- `Prefix` = `/varasto/`. Can be anything you like, but I recommend a prefix if you ever get
  any non-Varasto files in the same bucket (access logs are a good example).

- `RegionId` looks like `eu-central-1`. This is the code for the region you selected,
  see: [region ids](https://docs.aws.amazon.com/general/latest/gr/s3.html).

- `AccessKeyId` and `AccessKeySecret` as you created them in IAM.


Bonus reading: ransomware protection
------------------------------------

Please read about [Ransomware protection](../../security/ransomware-protection/index.md) first.

S3 enables us to have file overwrite protection by restricting our access keys to not being
able to delete data. Our policy in this guide doesn't allow for data deletion.

Ransomware cannot delete data, but theoretically ransomware could overwrite data since AWS
S3 doesn't differentiate between creating new file and updating old file in its `PutObject`
operation. This is where file versioning steps in - if you enable
[versioning](https://docs.aws.amazon.com/AmazonS3/latest/dev/Versioning.html) for the bucket,
any file overwrites just create a new version, and the un-infected version of data can be
recovered by accessing the old versions of the data once a ransomware infection is
identified.
