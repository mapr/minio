package org.example.objectstore;

import com.amazonaws.auth.AWSCredentials;
import com.amazonaws.auth.BasicSessionCredentials;
import com.amazonaws.services.s3.AmazonS3;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.dataformat.xml.XmlMapper;
import okhttp3.FormBody;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.Response;

import java.io.IOException;

public class ObjectstoreLdap {
  private static final String USERNAME = "admin";
  private static final String PASSWORD = "abc@123";
  private static final String URL = "http://localhost:9000";
  private static final String BUCKET_NAME = "test";
  private static final String FILE_NAME = "file";

  public static void main(String[] args) throws IOException {
    String passwordWithoutSpecialCharacters = replaceSpecialCharacters(PASSWORD);
    System.out.println("Your password without special characters: " + passwordWithoutSpecialCharacters);

    AWSCredentials credentials = getTemporaryCredentials(URL, USERNAME, passwordWithoutSpecialCharacters);
    AmazonS3 s3 = Operations.getConnection(URL, credentials, true);

    Operations.demo(s3, BUCKET_NAME, FILE_NAME);
  }

  private static String replaceSpecialCharacters(String string) {
    StringBuilder result = new StringBuilder();
    for (char val : string.toCharArray()) {
      if ((val >= 'a' && val <= 'z') || (val >= 'A' && val <= 'Z') || (val >= '0' && val <= '9'))
        result.append(val);
      else
        result.append("%").append(Integer.toHexString(val));
    }

      return result.toString();
  }

  private static BasicSessionCredentials getTemporaryCredentials(String url, String username, String password)
    throws IOException {
    String requestUrl = new StringBuilder()
      .append(url)
      .append("?Action=AssumeRoleWithLDAPIdentity")
      .append("&LDAPUsername=")
      .append(username)
      .append("&LDAPPassword=")
      .append(password)
      .append("&Version=2011-06-15")
      .toString();

    System.out.println("POST to " + requestUrl);

    Request request = new Request.Builder()
      .url(requestUrl)
      .addHeader("User-Agent", "OkHttp Bot")
      .post(new FormBody.Builder().build())
      .build();

    OkHttpClient httpClient = new OkHttpClient();
    byte[] result;
    try (Response response = httpClient.newCall(request).execute()) {

      if (!response.isSuccessful()) throw new IOException("Unexpected code " + response);

      result = response.body().bytes();
    }

    return parseCredentials(result);
  }

  private static BasicSessionCredentials parseCredentials(byte[] response) throws IOException {
    XmlMapper mapper = new XmlMapper();

    JsonNode node = mapper.readTree(response);
    node = node.findValue("AssumeRoleWithLDAPIdentityResult").findValue("Credentials");

    String accessKey = node.findValue("AccessKeyId").asText();
    String secretKey = node.findValue("SecretAccessKey").asText();
    String sessionId = node.findValue("SessionToken").asText();

    System.out.println();
    System.out.println("accessKey: " + accessKey);
    System.out.println("secretKey: " + secretKey);
    System.out.println("sessionToken: " + sessionId);

    return new BasicSessionCredentials(accessKey, secretKey, sessionId);
  }
}

