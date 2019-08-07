Setting up S3
=============


Create S3 bucket in AWS
-----------------------

Create a bucket, I named mine `function61-varasto-test`. Choose the region carefully (can affect
pricing and latency), as you can't easily change it later.

You can create the bucket with pretty much the default options.


Create access credentials
-------------------------

Move over to [IAM](https://console.aws.amazon.com/iam/home) to create restricted access
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
            "Resource": "arn:aws:s3:::function61-varasto-test/*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListBucket"
            ],
            "Resource": "arn:aws:s3:::function61-varasto-test*"
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

Don't worry too much about defining the limit, since you can easily change the quota later.


Mount the S3 bucket as volume in Varasto
----------------------------------------

Now mount the volume in Varasto. For kind, use `aws-s3`.

For driver options, write it as `bucket:regionId:accessKeyId:secret`.

If your details were these:

- `bucket = function61-varasto-test`

- `regionId = eu-central-1` (see: [region ids](https://docs.aws.amazon.com/general/latest/gr/rande.html))

- `accessKeyId = AKIAUZHTE3U35WCD5EHB`

- `secret = wXQJhB...`

Then your driver options would be `function61-varasto-test:eu-central-1:AKIAUZHTE3U35WCD5EHB:wXQJhB...`
