// @generated by protoc-gen-es v1.10.0
// @generated from file athletes/v1/athletes.proto (package athletes.v1, syntax proto3)
/* eslint-disable */
// @ts-nocheck

import { proto3, Timestamp } from "@bufbuild/protobuf";

/**
 * @generated from message athletes.v1.Athlete
 */
export const Athlete = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.Athlete",
  () => [
    { no: 1, name: "id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "create_time", kind: "message", T: Timestamp },
  ],
);

/**
 * @generated from message athletes.v1.CreateAthleteRequest
 */
export const CreateAthleteRequest = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.CreateAthleteRequest",
  () => [
    { no: 1, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ],
);

/**
 * @generated from message athletes.v1.CreateAthleteResponse
 */
export const CreateAthleteResponse = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.CreateAthleteResponse",
  () => [
    { no: 1, name: "athlete", kind: "message", T: Athlete },
  ],
);

/**
 * @generated from message athletes.v1.Activity
 */
export const Activity = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.Activity",
  () => [
    { no: 1, name: "id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "athlete_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 4, name: "create_time", kind: "message", T: Timestamp },
  ],
);

/**
 * @generated from message athletes.v1.CreateActivityRequest
 */
export const CreateActivityRequest = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.CreateActivityRequest",
  () => [
    { no: 1, name: "name", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "athlete_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
  ],
);

/**
 * @generated from message athletes.v1.CreateActivityResponse
 */
export const CreateActivityResponse = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.CreateActivityResponse",
  () => [
    { no: 1, name: "activity", kind: "message", T: Activity },
  ],
);

/**
 * @generated from message athletes.v1.BloodLactateMeasure
 */
export const BloodLactateMeasure = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.BloodLactateMeasure",
  () => [
    { no: 1, name: "id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "activity_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "mmol_per_liter", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 4, name: "heart_rate_bpm", kind: "scalar", T: 5 /* ScalarType.INT32 */ },
    { no: 5, name: "create_time", kind: "message", T: Timestamp },
  ],
);

/**
 * @generated from message athletes.v1.CreateBloodLactateMeasureRequest
 */
export const CreateBloodLactateMeasureRequest = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.CreateBloodLactateMeasureRequest",
  () => [
    { no: 1, name: "activity_id", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 2, name: "mmol_per_liter", kind: "scalar", T: 9 /* ScalarType.STRING */ },
    { no: 3, name: "heart_rate_bpm", kind: "scalar", T: 5 /* ScalarType.INT32 */ },
  ],
);

/**
 * @generated from message athletes.v1.CreateBloodLactateMeasureResponse
 */
export const CreateBloodLactateMeasureResponse = /*@__PURE__*/ proto3.makeMessageType(
  "athletes.v1.CreateBloodLactateMeasureResponse",
  () => [
    { no: 1, name: "blood_lactate_measure", kind: "message", T: BloodLactateMeasure },
  ],
);
