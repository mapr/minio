package org.example.objectstore;

import com.amazonaws.auth.AWSCredentials;
import com.amazonaws.auth.BasicAWSCredentials;
import com.amazonaws.services.s3.AmazonS3;

public class Objectstore {
  private static final String USERNAME = "minioadmin";
  private static final String PASSWORD = "minioadmin";
  private static final String URL = "http://localhost:9000";
  private static final String BUCKET_NAME = "test";
  private static final String FILE_NAME = "file";

  public static void main(String[] args) {
    AWSCredentials credentials = new BasicAWSCredentials(USERNAME, PASSWORD);
    AmazonS3 s3 = Operations.getConnection(URL, credentials, true);

    Operations.demo(s3, BUCKET_NAME, FILE_NAME);
  }
}
