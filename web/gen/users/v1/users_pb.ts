// @generated by protoc-gen-es v2.2.2 with parameter "target=ts"
// @generated from file users/v1/users.proto (package users.v1, syntax proto3)
/* eslint-disable */

import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv1";
import { fileDesc, messageDesc, serviceDesc } from "@bufbuild/protobuf/codegenv1";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file users/v1/users.proto.
 */
export const file_users_v1_users: GenFile = /*@__PURE__*/
  fileDesc("ChR1c2Vycy92MS91c2Vycy5wcm90bxIIdXNlcnMudjEibAoEVXNlchIKCgJpZBgBIAEoCRINCgVlbWFpbBgCIAEoCRIMCgRuYW1lGAMgASgJEhIKCmdpdmVuX25hbWUYBCABKAkSEwoLZmFtaWx5X25hbWUYBSABKAkSEgoKYXZhdGFyX3VybBgGIAEoCSIXChVHZXRDdXJyZW50VXNlclJlcXVlc3QiNgoWR2V0Q3VycmVudFVzZXJSZXNwb25zZRIcCgR1c2VyGAEgASgLMg4udXNlcnMudjEuVXNlcjJiCgtVc2VyU2VydmljZRJTCg5HZXRDdXJyZW50VXNlchIfLnVzZXJzLnYxLkdldEN1cnJlbnRVc2VyUmVxdWVzdBogLnVzZXJzLnYxLkdldEN1cnJlbnRVc2VyUmVzcG9uc2VCkQEKDGNvbS51c2Vycy52MUIKVXNlcnNQcm90b1ABWjRnaXRodWIuY29tL2NhdHprb3JuL3RyYWlsLXRvb2xzL2dlbi91c2Vycy92MTt1c2Vyc3YxogIDVVhYqgIIVXNlcnMuVjHKAghVc2Vyc1xWMeICFFVzZXJzXFYxXEdQQk1ldGFkYXRh6gIJVXNlcnM6OlYxYgZwcm90bzM");

/**
 * @generated from message users.v1.User
 */
export type User = Message<"users.v1.User"> & {
  /**
   * @generated from field: string id = 1;
   */
  id: string;

  /**
   * @generated from field: string email = 2;
   */
  email: string;

  /**
   * @generated from field: string name = 3;
   */
  name: string;

  /**
   * @generated from field: string given_name = 4;
   */
  givenName: string;

  /**
   * @generated from field: string family_name = 5;
   */
  familyName: string;

  /**
   * @generated from field: string avatar_url = 6;
   */
  avatarUrl: string;
};

/**
 * Describes the message users.v1.User.
 * Use `create(UserSchema)` to create a new message.
 */
export const UserSchema: GenMessage<User> = /*@__PURE__*/
  messageDesc(file_users_v1_users, 0);

/**
 * @generated from message users.v1.GetCurrentUserRequest
 */
export type GetCurrentUserRequest = Message<"users.v1.GetCurrentUserRequest"> & {
};

/**
 * Describes the message users.v1.GetCurrentUserRequest.
 * Use `create(GetCurrentUserRequestSchema)` to create a new message.
 */
export const GetCurrentUserRequestSchema: GenMessage<GetCurrentUserRequest> = /*@__PURE__*/
  messageDesc(file_users_v1_users, 1);

/**
 * @generated from message users.v1.GetCurrentUserResponse
 */
export type GetCurrentUserResponse = Message<"users.v1.GetCurrentUserResponse"> & {
  /**
   * @generated from field: users.v1.User user = 1;
   */
  user?: User;
};

/**
 * Describes the message users.v1.GetCurrentUserResponse.
 * Use `create(GetCurrentUserResponseSchema)` to create a new message.
 */
export const GetCurrentUserResponseSchema: GenMessage<GetCurrentUserResponse> = /*@__PURE__*/
  messageDesc(file_users_v1_users, 2);

/**
 * @generated from service users.v1.UserService
 */
export const UserService: GenService<{
  /**
   * @generated from rpc users.v1.UserService.GetCurrentUser
   */
  getCurrentUser: {
    methodKind: "unary";
    input: typeof GetCurrentUserRequestSchema;
    output: typeof GetCurrentUserResponseSchema;
  },
}> = /*@__PURE__*/
  serviceDesc(file_users_v1_users, 0);
