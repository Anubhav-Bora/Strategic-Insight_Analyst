"use client";

import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import Link from "next/link";
import { FileUp, Plus } from "lucide-react";
import api from "@/lib/api";
import { useAuthContext } from "@/context/AuthContext";
import { redirect } from "next/navigation";
import { format } from "date-fns";
import { getAuth } from "firebase/auth";

interface Document {
  id: string;
  fileName: string;
  storageUrl: string;
  uploadedAt: string;
}

export default function DashboardPage() {
  const { user } = useAuthContext();

  if (!user) {
    redirect("/login");
  }

  const auth = getAuth();
  const { data: documents, isLoading } = useQuery<Document[]>({
    queryKey: ["documents"],
    queryFn: async () => {
      const user = auth.currentUser;
      if (!user) throw new Error("Not authenticated");
      const token = await user.getIdToken();
      const { data } = await api.get("/api/documents", {
        headers: { Authorization: `Bearer ${token}` },
      });
      return data;
    },
  });

  return (
    <div className="container mx-auto py-8">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Your Documents</h1>
        <Link href="/upload">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Upload Document
          </Button>
        </Link>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Skeleton key={i} className="h-32 w-full" />
          ))}
        </div>
      ) : documents && documents.length > 0 ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {documents.map((doc) => (
            <Link key={doc.id} href={`/documents/${doc.id}`}>
              <Card className="h-full p-4 transition-colors hover:bg-gray-50">
                <div className="flex items-center gap-3">
                  <div className="rounded-lg bg-blue-100 p-3">
                    <FileUp className="h-5 w-5 text-blue-600" />
                  </div>
                  <div>
                    <h3 className="font-medium">{doc.fileName}</h3>
                    <p className="text-sm text-gray-500">
                      {format(new Date(doc.uploadedAt), "MMM dd, yyyy")}
                    </p>
                  </div>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      ) : (
        <Card className="p-8 text-center">
          <div className="mx-auto max-w-md">
            <FileUp className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-4 text-lg font-medium">No documents yet</h3>
            <p className="mt-2 text-gray-500">
              Upload your first document to get started with analysis
            </p>
            <Link href="/upload">
              <Button className="mt-4">
                <Plus className="mr-2 h-4 w-4" />
                Upload Document
              </Button>
            </Link>
          </div>
        </Card>
      )}
    </div>
  );
}