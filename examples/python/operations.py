from datetime import datetime

from botocore.exceptions import ClientError


def demo(s3, bucket_name, file):
    print("\n")
    list_buckets(s3)

    print("\n")
    create_bucket_if_not_exists(s3, bucket_name)
    list_buckets(s3)

    print("\n")
    upload_file_if_not_exists(s3, bucket_name, file)
    list_folder(s3, bucket_name)

    print("\n")
    read_file(s3, bucket_name, file)

    print("\n")
    delete_all(s3, bucket_name)
    list_folder(s3, bucket_name)

    print("\n")
    delete_bucket_if_exists(s3, bucket_name)
    list_buckets(s3)

def list_buckets(s3):
    print("Buckets:")
    for bucket in s3.list_buckets()['Buckets']:
        print(bucket['Name'])


def create_bucket_if_not_exists(s3, bucket_name):
    if check_if_bucket_exists(s3, bucket_name):
        print("Bucket exists")
        return
    print("Bucket does not exist, creating...")
    s3.create_bucket(Bucket=bucket_name)


def check_if_bucket_exists(s3, bucket_name):
    print("Checking if bucket '" + bucket_name + "' exists")
    try:
        s3.head_bucket(Bucket=bucket_name)
    except ClientError as e:
        return int(e.response['Error']['Code']) != 404
    return True


def delete_bucket_if_exists(s3, bucket_name):
    print("Checking if bucket '" + bucket_name + "' exists")
    try:
        s3.head_bucket(Bucket=bucket_name)
        print("Bucket exists, deleting...")
        s3.delete_bucket(Bucket=bucket_name)
    except ClientError:
        print("Bucket does not exist")


def list_folder(s3, bucket_name):
    print("Listing bucket " + bucket_name + ":")
    objects = s3.list_objects(Bucket=bucket_name)
    if 'Contents' in objects:
        for key in objects['Contents']:
            print(key['Key'])
    else:
        print("Is empty")


def delete_all(s3, bucket_name):
    print("Cleaning all form bucket: " + bucket_name)
    objects = s3.list_objects(Bucket=bucket_name)
    if 'Contents' in objects:
        for key in objects['Contents']:
            s3.delete_object(Bucket=bucket_name, Key=key['Key'])


def upload_file_if_not_exists(s3, bucket_name, file_name):
    if check_if_file_exist(s3, bucket_name, file_name):
        print("File '" + file_name + "' already exists in bucket '" + bucket_name + "'")
        return

    date = datetime.now().strftime("%m/%d/%Y, %H:%M:%S")
    data = "Hello world on " + date + "!"

    print("Uploading '" + file_name + "' to '" + bucket_name + "' with data:\n" + data)
    s3.put_object(Body=data.encode('ascii'), Bucket=bucket_name, Key=file_name)


def read_file(s3, bucket_name, file_name):
    print("Reading file '" + file_name + "' from bucket '" + bucket_name + "'")
    obj = s3.get_object(Bucket=bucket_name, Key=file_name)
    print("Data:")
    print(obj['Body'].read().decode('utf-8'))


def check_if_file_exist(s3, bucket_name, file):
    print("Checking if file '" + file + "' exists in bucket '" + bucket_name + "'")
    try:
        s3.head_object(Bucket=bucket_name, Key=file)
    except ClientError as e:
        return int(e.response['Error']['Code']) != 404
    return True
