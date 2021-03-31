import boto3
import urllib3

from botocore.client import Config
from operations import demo


if __name__ == '__main__':
    host = "http://localhost:9000"
    username = "minioadmin"
    password = "minioadmin"
    bucketName = "test"
    file = "file"

    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    s3 = boto3.client('s3',
                      endpoint_url=host,
                      aws_access_key_id=username,
                      aws_secret_access_key=password,
                      config=Config(signature_version='s3v4'),
                      region_name='us-east-1',
                      verify=False)

    demo(s3, bucketName, file)

