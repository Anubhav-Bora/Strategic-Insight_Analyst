"use client";

import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { FileUp, Upload } from "lucide-react";
import { useToast } from "@/components/ui/toaster";
import api from "@/lib/api";
import { getAuth } from "firebase/auth";

const formSchema = z.object({
  file: z.any().refine((file) => file instanceof FileList && file.length === 1, "File is required"),
});

export default function UploadPage() {
  const router = useRouter();
  const toast = useToast();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const fileRef = form.register("file");

  async function onSubmit(values: z.infer<typeof formSchema>) {
    try {
      const auth = getAuth();
      const user = auth.currentUser;
      if (!user) {
        toast("You must be logged in to upload documents");
        return;
      }
      const token = await user.getIdToken();

      const formData = new FormData();
      formData.append("document", values.file[0]);

      const { data } = await api.post("/api/documents", formData, {
        headers: {
          "Content-Type": "multipart/form-data",
          "Authorization": `Bearer ${token}`,
        },
      });

      toast("Document uploaded: Your document has been uploaded successfully");
      router.push(`/documents/${data.id}`);
    } catch (error) {
      toast("Upload failed: " + (error instanceof Error ? error.message : "Unknown error"));
    }
  }

  return (
    <div className="container mx-auto py-8">
      <Card className="mx-auto max-w-2xl p-6">
        <div className="mb-6 flex items-center gap-3">
          <FileUp className="h-6 w-6 text-blue-600" />
          <h1 className="text-2xl font-bold">Upload Document</h1>
        </div>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            <FormField
              control={form.control}
              name="file"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Document</FormLabel>
                  <FormControl>
                    <div className="flex items-center gap-4">
                      <Input
                        type="file"
                        accept=".pdf,.txt"
                        {...fileRef}
                        onChange={(e) => {
                          field.onChange(e.target.files);
                        }}
                      />
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex justify-end">
              <Button type="submit">
                <Upload className="mr-2 h-4 w-4" />
                Upload
              </Button>
            </div>
          </form>
        </Form>
      </Card>
    </div>
  );
}