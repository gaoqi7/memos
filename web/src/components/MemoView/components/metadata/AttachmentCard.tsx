import { cn } from "@/lib/utils";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { getAttachmentThumbnailUrl, getAttachmentType, getAttachmentUrl } from "@/utils/attachment";

interface AttachmentCardProps {
  attachment: Attachment;
  onClick?: () => void;
  className?: string;
}

const AttachmentCard = ({ attachment, onClick, className }: AttachmentCardProps) => {
  const attachmentType = getAttachmentType(attachment);
  const sourceUrl = getAttachmentUrl(attachment);

  if (attachmentType === "image/*") {
    return (
      <img
        src={sourceUrl}
        alt={attachment.filename}
        className={cn("w-full h-full object-cover rounded-lg cursor-pointer", className)}
        onClick={onClick}
        loading="lazy"
      />
    );
  }

  if (attachmentType === "video/*") {
    // When an onClick handler is provided, the video opens in a lightbox.
    // Show a poster thumbnail instead of loading video data.
    if (onClick) {
      const posterUrl = attachment.immichAssetId
        ? `/file/immich/${attachment.immichAssetId}?size=thumbnail`
        : getAttachmentThumbnailUrl(attachment);

      return (
        <video
          src={sourceUrl}
          poster={posterUrl}
          className={cn("w-full h-full object-cover rounded-lg pointer-events-none", className)}
          playsInline
          preload="none"
          muted
          onPlay={(e) => e.currentTarget.pause()}
        />
      );
    }

    return (
      <video
        src={sourceUrl}
        className={cn("w-full h-full object-cover rounded-lg", className)}
        controls
        playsInline
        preload="metadata"
      />
    );
  }

  return null;
};

export default AttachmentCard;
