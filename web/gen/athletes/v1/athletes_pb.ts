// @generated by protoc-gen-es v2.2.2 with parameter "target=ts"
// @generated from file athletes/v1/athletes.proto (package athletes.v1, syntax proto3)
/* eslint-disable */

import type { GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv1";
import { fileDesc, messageDesc, serviceDesc } from "@bufbuild/protobuf/codegenv1";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { file_google_protobuf_timestamp } from "@bufbuild/protobuf/wkt";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file athletes/v1/athletes.proto.
 */
export const file_athletes_v1_athletes: GenFile = /*@__PURE__*/
  fileDesc("ChphdGhsZXRlcy92MS9hdGhsZXRlcy5wcm90bxILYXRobGV0ZXMudjEiVAoHQXRobGV0ZRIKCgJpZBgBIAEoCRIMCgRuYW1lGAIgASgJEi8KC2NyZWF0ZV90aW1lGAMgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcCIkChRDcmVhdGVBdGhsZXRlUmVxdWVzdBIMCgRuYW1lGAEgASgJIj4KFUNyZWF0ZUF0aGxldGVSZXNwb25zZRIlCgdhdGhsZXRlGAEgASgLMhQuYXRobGV0ZXMudjEuQXRobGV0ZSJpCghBY3Rpdml0eRIKCgJpZBgBIAEoCRIMCgRuYW1lGAIgASgJEhIKCmF0aGxldGVfaWQYAyABKAkSLwoLY3JlYXRlX3RpbWUYBCABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wIjkKFUNyZWF0ZUFjdGl2aXR5UmVxdWVzdBIMCgRuYW1lGAEgASgJEhIKCmF0aGxldGVfaWQYAiABKAkiQQoWQ3JlYXRlQWN0aXZpdHlSZXNwb25zZRInCghhY3Rpdml0eRgBIAEoCzIVLmF0aGxldGVzLnYxLkFjdGl2aXR5IpcBChNCbG9vZExhY3RhdGVNZWFzdXJlEgoKAmlkGAEgASgJEhMKC2FjdGl2aXR5X2lkGAIgASgJEhYKDm1tb2xfcGVyX2xpdGVyGAMgASgJEhYKDmhlYXJ0X3JhdGVfYnBtGAQgASgFEi8KC2NyZWF0ZV90aW1lGAUgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcCJnCiBDcmVhdGVCbG9vZExhY3RhdGVNZWFzdXJlUmVxdWVzdBITCgthY3Rpdml0eV9pZBgBIAEoCRIWCg5tbW9sX3Blcl9saXRlchgCIAEoCRIWCg5oZWFydF9yYXRlX2JwbRgDIAEoBSJkCiFDcmVhdGVCbG9vZExhY3RhdGVNZWFzdXJlUmVzcG9uc2USPwoVYmxvb2RfbGFjdGF0ZV9tZWFzdXJlGAEgASgLMiAuYXRobGV0ZXMudjEuQmxvb2RMYWN0YXRlTWVhc3VyZTK/AgoOQXRobGV0ZVNlcnZpY2USVgoNQ3JlYXRlQXRobGV0ZRIhLmF0aGxldGVzLnYxLkNyZWF0ZUF0aGxldGVSZXF1ZXN0GiIuYXRobGV0ZXMudjEuQ3JlYXRlQXRobGV0ZVJlc3BvbnNlElkKDkNyZWF0ZUFjdGl2aXR5EiIuYXRobGV0ZXMudjEuQ3JlYXRlQWN0aXZpdHlSZXF1ZXN0GiMuYXRobGV0ZXMudjEuQ3JlYXRlQWN0aXZpdHlSZXNwb25zZRJ6ChlDcmVhdGVCbG9vZExhY3RhdGVNZWFzdXJlEi0uYXRobGV0ZXMudjEuQ3JlYXRlQmxvb2RMYWN0YXRlTWVhc3VyZVJlcXVlc3QaLi5hdGhsZXRlcy52MS5DcmVhdGVCbG9vZExhY3RhdGVNZWFzdXJlUmVzcG9uc2VCsgEKD2NvbS5hdGhsZXRlcy52MUINQXRobGV0ZXNQcm90b1ABWkNnaXRodWIuY29tL2NhdHprb3JuL3RyYWlsLXRvb2xzL2ludGVybmFsL2dlbi9hdGhsZXRlcy92MTthdGhsZXRlc3YxogIDQVhYqgILQXRobGV0ZXMuVjHKAgtBdGhsZXRlc1xWMeICF0F0aGxldGVzXFYxXEdQQk1ldGFkYXRh6gIMQXRobGV0ZXM6OlYxYgZwcm90bzM", [file_google_protobuf_timestamp]);

/**
 * @generated from message athletes.v1.Athlete
 */
export type Athlete = Message<"athletes.v1.Athlete"> & {
  /**
   * @generated from field: string id = 1;
   */
  id: string;

  /**
   * @generated from field: string name = 2;
   */
  name: string;

  /**
   * @generated from field: google.protobuf.Timestamp create_time = 3;
   */
  createTime?: Timestamp;
};

/**
 * Describes the message athletes.v1.Athlete.
 * Use `create(AthleteSchema)` to create a new message.
 */
export const AthleteSchema: GenMessage<Athlete> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 0);

/**
 * @generated from message athletes.v1.CreateAthleteRequest
 */
export type CreateAthleteRequest = Message<"athletes.v1.CreateAthleteRequest"> & {
  /**
   * @generated from field: string name = 1;
   */
  name: string;
};

/**
 * Describes the message athletes.v1.CreateAthleteRequest.
 * Use `create(CreateAthleteRequestSchema)` to create a new message.
 */
export const CreateAthleteRequestSchema: GenMessage<CreateAthleteRequest> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 1);

/**
 * @generated from message athletes.v1.CreateAthleteResponse
 */
export type CreateAthleteResponse = Message<"athletes.v1.CreateAthleteResponse"> & {
  /**
   * @generated from field: athletes.v1.Athlete athlete = 1;
   */
  athlete?: Athlete;
};

/**
 * Describes the message athletes.v1.CreateAthleteResponse.
 * Use `create(CreateAthleteResponseSchema)` to create a new message.
 */
export const CreateAthleteResponseSchema: GenMessage<CreateAthleteResponse> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 2);

/**
 * @generated from message athletes.v1.Activity
 */
export type Activity = Message<"athletes.v1.Activity"> & {
  /**
   * @generated from field: string id = 1;
   */
  id: string;

  /**
   * @generated from field: string name = 2;
   */
  name: string;

  /**
   * @generated from field: string athlete_id = 3;
   */
  athleteId: string;

  /**
   * @generated from field: google.protobuf.Timestamp create_time = 4;
   */
  createTime?: Timestamp;
};

/**
 * Describes the message athletes.v1.Activity.
 * Use `create(ActivitySchema)` to create a new message.
 */
export const ActivitySchema: GenMessage<Activity> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 3);

/**
 * @generated from message athletes.v1.CreateActivityRequest
 */
export type CreateActivityRequest = Message<"athletes.v1.CreateActivityRequest"> & {
  /**
   * @generated from field: string name = 1;
   */
  name: string;

  /**
   * @generated from field: string athlete_id = 2;
   */
  athleteId: string;
};

/**
 * Describes the message athletes.v1.CreateActivityRequest.
 * Use `create(CreateActivityRequestSchema)` to create a new message.
 */
export const CreateActivityRequestSchema: GenMessage<CreateActivityRequest> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 4);

/**
 * @generated from message athletes.v1.CreateActivityResponse
 */
export type CreateActivityResponse = Message<"athletes.v1.CreateActivityResponse"> & {
  /**
   * @generated from field: athletes.v1.Activity activity = 1;
   */
  activity?: Activity;
};

/**
 * Describes the message athletes.v1.CreateActivityResponse.
 * Use `create(CreateActivityResponseSchema)` to create a new message.
 */
export const CreateActivityResponseSchema: GenMessage<CreateActivityResponse> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 5);

/**
 * @generated from message athletes.v1.BloodLactateMeasure
 */
export type BloodLactateMeasure = Message<"athletes.v1.BloodLactateMeasure"> & {
  /**
   * @generated from field: string id = 1;
   */
  id: string;

  /**
   * @generated from field: string activity_id = 2;
   */
  activityId: string;

  /**
   * @generated from field: string mmol_per_liter = 3;
   */
  mmolPerLiter: string;

  /**
   * @generated from field: int32 heart_rate_bpm = 4;
   */
  heartRateBpm: number;

  /**
   * @generated from field: google.protobuf.Timestamp create_time = 5;
   */
  createTime?: Timestamp;
};

/**
 * Describes the message athletes.v1.BloodLactateMeasure.
 * Use `create(BloodLactateMeasureSchema)` to create a new message.
 */
export const BloodLactateMeasureSchema: GenMessage<BloodLactateMeasure> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 6);

/**
 * @generated from message athletes.v1.CreateBloodLactateMeasureRequest
 */
export type CreateBloodLactateMeasureRequest = Message<"athletes.v1.CreateBloodLactateMeasureRequest"> & {
  /**
   * @generated from field: string activity_id = 1;
   */
  activityId: string;

  /**
   * @generated from field: string mmol_per_liter = 2;
   */
  mmolPerLiter: string;

  /**
   * @generated from field: int32 heart_rate_bpm = 3;
   */
  heartRateBpm: number;
};

/**
 * Describes the message athletes.v1.CreateBloodLactateMeasureRequest.
 * Use `create(CreateBloodLactateMeasureRequestSchema)` to create a new message.
 */
export const CreateBloodLactateMeasureRequestSchema: GenMessage<CreateBloodLactateMeasureRequest> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 7);

/**
 * @generated from message athletes.v1.CreateBloodLactateMeasureResponse
 */
export type CreateBloodLactateMeasureResponse = Message<"athletes.v1.CreateBloodLactateMeasureResponse"> & {
  /**
   * @generated from field: athletes.v1.BloodLactateMeasure blood_lactate_measure = 1;
   */
  bloodLactateMeasure?: BloodLactateMeasure;
};

/**
 * Describes the message athletes.v1.CreateBloodLactateMeasureResponse.
 * Use `create(CreateBloodLactateMeasureResponseSchema)` to create a new message.
 */
export const CreateBloodLactateMeasureResponseSchema: GenMessage<CreateBloodLactateMeasureResponse> = /*@__PURE__*/
  messageDesc(file_athletes_v1_athletes, 8);

/**
 * @generated from service athletes.v1.AthleteService
 */
export const AthleteService: GenService<{
  /**
   * @generated from rpc athletes.v1.AthleteService.CreateAthlete
   */
  createAthlete: {
    methodKind: "unary";
    input: typeof CreateAthleteRequestSchema;
    output: typeof CreateAthleteResponseSchema;
  },
  /**
   * @generated from rpc athletes.v1.AthleteService.CreateActivity
   */
  createActivity: {
    methodKind: "unary";
    input: typeof CreateActivityRequestSchema;
    output: typeof CreateActivityResponseSchema;
  },
  /**
   * @generated from rpc athletes.v1.AthleteService.CreateBloodLactateMeasure
   */
  createBloodLactateMeasure: {
    methodKind: "unary";
    input: typeof CreateBloodLactateMeasureRequestSchema;
    output: typeof CreateBloodLactateMeasureResponseSchema;
  },
}> = /*@__PURE__*/
  serviceDesc(file_athletes_v1_athletes, 0);

