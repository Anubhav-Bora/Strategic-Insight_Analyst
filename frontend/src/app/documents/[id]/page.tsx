"use client";

import { useParams } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { FileUp, Trash2 } from "lucide-react";
import api from "@/lib/api";
import { format } from "date-fns";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/toaster";
import DocumentChat from "@/components/DocumentChat";
import { Skeleton } from "@/components/ui/skeleton";
import { auth } from "@/lib/firebase";
import { useEffect, useState } from "react";

interface Document {
  id: string;
  fileName: string;
  storageUrl: string;
  uploadedAt: string;
}

export default function DocumentPage() {
  const { id } = useParams();
  const router = useRouter();
  const toast = useToast();
  const [previewContent, setPreviewContent] = useState<string | null>(null);
  const [isPdf, setIsPdf] = useState(false);

  const { data: document, isLoading } = useQuery<Document>({
    queryKey: ["document", id],
    queryFn: async () => {
      const user = auth.currentUser;
      if (!user) throw new Error("Not authenticated");
      const token = await user.getIdToken();
      const { data } = await api.get(`/api/documents/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      return data;
    },
  });

  const backendBaseUrl = process.env.NEXT_PUBLIC_API_URL?.replace(/\/api$/, "");
  const fileUrl = document?.storageUrl?.startsWith("http")
    ? document.storageUrl
    : `${backendBaseUrl}${document?.storageUrl || ""}`;

  useEffect(() => {
    if (fileUrl && fileUrl !== backendBaseUrl && !isPdf) {
      fetch(fileUrl)
        .then((res) => res.text())
        .then(setPreviewContent)
        .catch(() => setPreviewContent("Failed to load preview."));
    }
  }, [fileUrl, isPdf]);

  useEffect(() => {
    if (document?.storageUrl) {
      const url = document.storageUrl;
      setIsPdf(url.toLowerCase().endsWith(".pdf"));
    }
  }, [document?.storageUrl]);

  async function handleDelete() {
    try {
      const user = auth.currentUser;
      if (!user) throw new Error("Not authenticated");
      const token = await user.getIdToken();
      await api.delete(`/api/documents/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      toast("Document deleted: Your document has been deleted successfully");
      router.push("/");
    } catch (error) {
      toast("Delete failed: " + (error instanceof Error ? error.message : "Unknown error"));
    }
  }

  if (isLoading) {
    return (
      <div className="container mx-auto py-8">
        <div className="flex items-center justify-between">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-10 w-24" />
        </div>
        <Skeleton className="mt-8 h-96 w-full" />
      </div>
    );
  }

  if (!document) {
    return (
      <div className="container mx-auto py-8">
        <Card className="p-8 text-center">
          <h3 className="text-lg font-medium">Document not found</h3>
          <Link href="/">
            <Button className="mt-4">Back to documents</Button>
          </Link>
        </Card>
      </div>
    );
  }

  return (
    <div className="container mx-auto py-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{document.fileName}</h1>
          <p className="text-gray-500">
            Uploaded on {format(new Date(document.uploadedAt), "MMM dd, yyyy")}
          </p>
        </div>
        <Button variant="destructive" onClick={handleDelete}>
          <Trash2 className="mr-2 h-4 w-4" />
          Delete
        </Button>
      </div>

      <div className="mt-8 grid gap-8 lg:grid-cols-2">
        <Card className="p-6">
          <div className="mb-6 flex items-center gap-3">
            <FileUp className="h-5 w-5 text-blue-600" />
            <h2 className="text-xl font-semibold">Document Preview</h2>
          </div>
          <div className="h-96 overflow-auto rounded-lg border p-4">
            {!fileUrl || fileUrl === backendBaseUrl ? (
              <p className="text-gray-500">No preview available</p>
            ) : isPdf ? (
              <iframe
                src={fileUrl}
                width="100%"
                height="100%"
                style={{ minHeight: "22rem", border: "none" }}
                title="PDF Preview"
              />
            ) : previewContent !== null ? (
              <pre className="whitespace-pre-wrap">{previewContent}</pre>
            ) : (
              <p className="text-gray-500">Loading preview...</p>
            )}
          </div>
        </Card>

        <DocumentChat documentId={id as string} />
      </div>
    </div>
  );
}