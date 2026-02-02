import { create } from "@bufbuild/protobuf";
import { attachmentServiceClient } from "@/connect";
import { AttachmentSchema } from "@/types/proto/api/v1/attachment_service_pb";

export async function createExternalAttachment(externalLink: string) {
  return attachmentServiceClient.createAttachment({
    attachment: create(AttachmentSchema, {
      filename: "",
      type: "",
      externalLink,
    }),
  });
}
