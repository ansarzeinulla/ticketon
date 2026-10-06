"use client";

import { useRef, useState, type ChangeEvent } from "react";

import { Button } from "@/components/ui/button";
import { ApiError, api } from "@/lib/api";

/** Mirrors the server's limit, so an oversized file is refused before it is
 *  sent rather than after a slow upload. The server enforces it regardless. */
const MAX_BYTES = 5 * 1024 * 1024;

const ACCEPTED = "image/jpeg,image/png,image/gif,image/webp";

/**
 * Event banner upload (SRS 4.2, "add ... images").
 *
 * The component owns the upload and hands back a URL; the form owns the field.
 * That split means the same control works on the create form and on an edit
 * form later without either knowing how storage works.
 */
export function ImageUpload({
  value,
  onChange,
  disabled,
}: {
  value: string;
  onChange: (url: string) => void;
  disabled?: boolean;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    // Reset immediately so picking the same file twice still fires a change.
    event.target.value = "";
    if (!file) return;

    if (file.size > MAX_BYTES) {
      setError("That image is larger than 5 MB.");
      return;
    }

    setUploading(true);
    setError(null);
    try {
      const uploaded = await api.uploadImage(file);
      onChange(uploaded.url);
    } catch (cause) {
      setError(
        cause instanceof ApiError
          ? cause.message
          : "Could not upload that image. Try again.",
      );
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="space-y-2">
      <span className="block text-sm font-medium text-foreground">Event banner</span>

      {value ? (
        <div className="space-y-2">
          {/*
            A plain <img>, not next/image: the URL points at the API's upload
            route, which is not a configured image domain, and optimising a
            banner an organizer just uploaded buys nothing here.
          */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={value}
            alt="The event banner as attendees will see it"
            className="max-h-48 w-full rounded-lg border border-border-subtle object-cover"
          />
          <div className="flex gap-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={disabled || uploading}
              onClick={() => inputRef.current?.click()}
            >
              Replace
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={disabled || uploading}
              onClick={() => onChange("")}
            >
              Remove
            </Button>
          </div>
        </div>
      ) : (
        <div className="rounded-lg border border-dashed border-border-subtle px-4 py-6 text-center">
          <p className="text-sm text-foreground-muted">
            A wide photograph works best on the public page.
          </p>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            className="mt-3"
            loading={uploading}
            disabled={disabled}
            onClick={() => inputRef.current?.click()}
          >
            Choose an image
          </Button>
          <p className="mt-2 text-xs text-foreground-muted">
            JPEG, PNG, GIF or WebP · up to 5 MB
          </p>
        </div>
      )}

      {error && (
        <p className="text-xs text-danger" role="alert">
          {error}
        </p>
      )}

      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED}
        className="hidden"
        onChange={(event) => void handleFile(event)}
      />
    </div>
  );
}
