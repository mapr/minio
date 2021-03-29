import json

import boto3
import requests
import urllib3
import xmltodict
from botocore.client import Config

from operations import *


def get_credentials(host, username, password):
    url = host + "?Action=AssumeRoleWithLDAPIdentity" \
                 "&LDAPUsername=" + username + \
          "&LDAPPassword=" + password + "&Version=2011-06-15"
    print("POST to " + url)
    response = requests.post(url)
    data = response.text
    data = xmltodict.parse(data)
    data = json.dumps(data)
    data = json.loads(data)
    credentials = data["AssumeRoleWithLDAPIdentityResponse"]["AssumeRoleWithLDAPIdentityResult"]["Credentials"]
    access = credentials["AccessKeyId"]
    secret = credentials["SecretAccessKey"]
    session = credentials["SessionToken"]
    return access, secret, session


def replace_special_characters(string):
    alphanumeric = ""
    for character in string:
        if character.isalnum():
            alphanumeric += character
        else:
            alphanumeric += "%" + str(format(ord(character), "x"))

    return alphanumeric


if __name__ == '__main__':
    host = "http://localhost:9000"
    username = "admin"
    password = "abc@123"
    bucketName = "test"
    file = "file"

    passwordWithoutSpecialCharacters = replace_special_characters(password)
    print("Your password without special characters: " + passwordWithoutSpecialCharacters)

    accessKey, secretKey, sessionToken = get_credentials(host, username, passwordWithoutSpecialCharacters)
    print("\n")
    print("accessKey: " + accessKey)
    print("secretKey: " + secretKey)
    print("sessionToken: " + sessionToken)

    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    s3 = boto3.client('s3',
                      endpoint_url=host,
                      aws_access_key_id=accessKey,
                      aws_secret_access_key=secretKey,
                      aws_session_token=sessionToken,
                      config=Config(signature_version='s3v4'),
                      region_name='us-east-1',
                      verify=False)

    demo(s3, bucketName, file)
