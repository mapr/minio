package org.example.objectstore;

import com.amazonaws.ClientConfiguration;
import com.amazonaws.SDKGlobalConfiguration;
import com.amazonaws.auth.AWSCredentials;
import com.amazonaws.auth.AWSStaticCredentialsProvider;
import com.amazonaws.client.builder.AwsClientBuilder;
import com.amazonaws.regions.Regions;
import com.amazonaws.services.s3.AmazonS3;
import com.amazonaws.services.s3.AmazonS3ClientBuilder;
import com.amazonaws.services.s3.model.Bucket;
import com.amazonaws.services.s3.model.ObjectListing;
import com.amazonaws.services.s3.model.ObjectMetadata;
import com.amazonaws.services.s3.model.PutObjectRequest;
import com.amazonaws.services.s3.model.S3Object;
import com.amazonaws.services.s3.model.S3ObjectSummary;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.List;
import java.util.stream.Collectors;

public class Operations {
  private static final String S3_SIGNER = "S3SignerType";
  private static final String S3_V4_SIGNER = "AWSS3V4SignerType";

  public static AmazonS3 getConnection(String url, AWSCredentials credentials, boolean isS3v4) {
      final ClientConfiguration clientConfiguration = new ClientConfiguration();
      clientConfiguration.setSignerOverride(isS3v4 ? S3_V4_SIGNER : S3_SIGNER);

      System.setProperty(SDKGlobalConfiguration.DISABLE_CERT_CHECKING_SYSTEM_PROPERTY, "true");

      return AmazonS3ClientBuilder
        .standard()
        .withEndpointConfiguration(new AwsClientBuilder.EndpointConfiguration(url, Regions.US_EAST_1.name()))
        .withPathStyleAccessEnabled(true)
        .withClientConfiguration(clientConfiguration)
        .withCredentials(new AWSStaticCredentialsProvider(credentials))
        .build();
  }

  public static void demo(AmazonS3 s3, String bucketName, String file) {
    System.out.println();
    Operations.listBuckets(s3);

    System.out.println();
    Operations.createBucketIfNotExists(s3, bucketName);
    Operations.listBuckets(s3);

    System.out.println();
    Operations.uploadFileIfNotExists(s3, bucketName, file);
    Operations.listFolder(s3, bucketName);

    System.out.println();
    Operations.readFile(s3, bucketName, file);

    System.out.println();
    Operations.clearBucket(s3, bucketName);
    Operations.listFolder(s3, bucketName);

    System.out.println();
    Operations.deleteBucketIfExists(s3, bucketName);
    Operations.listBuckets(s3);
  }

  public static void listBuckets(AmazonS3 s3) {
    List<Bucket> buckets = s3.listBuckets();
    System.out.println("Buckets:");
    buckets.stream().map(Bucket::getName).forEach(System.out::println);
  }

  public static boolean checkIfBucketExists(AmazonS3 s3, String bucketName) {
    System.out.println("Checking if bucket '" + bucketName + "' exists...");
    return s3.doesBucketExistV2(bucketName);
  }

  public static void createBucketIfNotExists(AmazonS3 s3, String bucketName) {
    if (checkIfBucketExists(s3, bucketName)) {
      System.out.println("Bucket '" + bucketName + "' exists");
      return;
    }

    System.out.println("Creating bucket '" + bucketName + "' ...");
    s3.createBucket(bucketName);
  }

  public static void deleteBucketIfExists(AmazonS3 s3, String bucketName) {
    if (!checkIfBucketExists(s3, bucketName)) {
      System.out.println("Bucket '" + bucketName + "' not exists");
      return;
    }

    System.out.println("Deleting bucket '" + bucketName + "' ...");
    s3.deleteBucket(bucketName);
  }

  public static boolean checkIfFileExists(AmazonS3 s3, String bucketName, String file) {
    System.out.println("Checking if file '" + file + "' exists in bucket '" + bucketName + "'");
    return s3.doesObjectExist(bucketName, file);
  }

  public static void uploadFileIfNotExists(AmazonS3 s3, String bucketName, String file) {
    if (checkIfFileExists(s3, bucketName, file)) {
      System.out.println("File '" + file + "' exists in bucket '" + bucketName + "'");
      return;
    }

    SimpleDateFormat formatter= new SimpleDateFormat("MM/dd/yyyy 'at' HH:mm:ss");
    Date date = new Date(System.currentTimeMillis());
    String data = "Hello world on " + formatter.format(date) + "!";
    System.out.println("Uploading '" + file + "' to '" + bucketName + "' with data:\n" + data + "\n");

    ObjectMetadata metadata = new ObjectMetadata();
    metadata.setContentType("plain/text");

    InputStream dataStream = new ByteArrayInputStream(data.getBytes());
    PutObjectRequest request = new PutObjectRequest(bucketName, file, dataStream, metadata);

    s3.putObject(request);
  }

  public static void listFolder(AmazonS3 s3, String bucketName) {
    System.out.println("Listing bucket '" + bucketName +"':");
    ObjectListing listing = s3.listObjects(bucketName);
    listing.getObjectSummaries()
      .stream()
      .map(S3ObjectSummary::getKey)
      .forEach(System.out::println);
  }

  public static void clearBucket(AmazonS3 s3, String bucketName) {
    System.out.println("Cleaning all from bucket '" + bucketName +"'..");
    ObjectListing listing = s3.listObjects(bucketName);
    listing.getObjectSummaries()
      .stream()
      .map(S3ObjectSummary::getKey)
      .forEach(file -> s3.deleteObject(bucketName, file));
  }

  public static void readFile(AmazonS3 s3, String bucketName, String file) {
    System.out.println("Reading file '" + file +"' from bucket '" + bucketName + "'");
    S3Object object = s3.getObject(bucketName, file);
    String data = new BufferedReader(new InputStreamReader(object.getObjectContent(), StandardCharsets.UTF_8))
        .lines()
      .collect(Collectors.joining("\n"));
    System.out.println("Data:\n" + data);
  }
}
