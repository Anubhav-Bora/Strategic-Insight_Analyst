"use client"
import { useParams } from "next/navigation"
import { useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { FileUp, Trash2, Download, Share2, Calendar, FileText } from "lucide-react"
import api from "@/lib/api"
import { format } from "date-fns"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useToast } from "@/components/ui/toaster"
import DocumentChat from "@/components/DocumentChat"
import { Skeleton } from "@/components/ui/skeleton"
import { auth } from "@/lib/firebase"
import { useEffect, useState } from "react"
import { motion } from "framer-motion"

interface Document {
  id: string
  fileName: string
  storageUrl: string
  uploadedAt: string
}

const fadeInUp = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.5, ease: "easeOut" }
}

export default function DocumentPage() {
  const { id } = useParams()
  const router = useRouter()
  const toast = useToast()
  const [previewContent, setPreviewContent] = useState<string | null>(null)
  const [isPdf, setIsPdf] = useState(false)

  const { data: document, isLoading } = useQuery<Document>({
    queryKey: ["document", id],
    queryFn: async () => {
      const user = auth.currentUser
      if (!user) throw new Error("Not authenticated")
      const token = await user.getIdToken()
      const { data } = await api.get(`/api/documents/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      return data
    },
  })

  const backendBaseUrl = process.env.NEXT_PUBLIC_API_URL?.replace(/\/api$/, "")
  const fileUrl = document?.storageUrl?.startsWith("http")
    ? document.storageUrl
    : `${backendBaseUrl}${document?.storageUrl || ""}`

  useEffect(() => {
    if (fileUrl && fileUrl !== backendBaseUrl && !isPdf) {
      fetch(fileUrl)
        .then((res) => res.text())
        .then(setPreviewContent)
        .catch(() => setPreviewContent("Failed to load preview."))
    }
  }, [fileUrl, isPdf])

  useEffect(() => {
    if (document?.storageUrl) {
      const url = document.storageUrl
      setIsPdf(url.toLowerCase().endsWith(".pdf"))
    }
  }, [document?.storageUrl])

  async function handleDelete() {
    try {
      const user = auth.currentUser
      if (!user) throw new Error("Not authenticated")
      const token = await user.getIdToken()
      await api.delete(`/api/documents/${id}`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      toast.toast({ title: "Document deleted successfully!" })
      router.push("/dashboard")
    } catch (error) {
      toast.toast({ title: "Failed to delete document: " + (error instanceof Error ? error.message : "Unknown error") })
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-background">
        <div className="container mx-auto py-8 px-4">
          <div className="flex items-center justify-between mb-8">
            <div className="space-y-2">
              <Skeleton className="h-8 w-64" />
              <Skeleton className="h-4 w-48" />
            </div>
            <Skeleton className="h-10 w-24" />
          </div>
          <div className="grid lg:grid-cols-2 gap-8">
            <Skeleton className="h-96 w-full" />
            <Skeleton className="h-96 w-full" />
          </div>
        </div>
      </div>
    )
  }

  if (!document) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <Card className="p-8 text-center max-w-md mx-auto">
          <div className="w-16 h-16 bg-gradient-to-r from-red-500 to-pink-500 rounded-2xl flex items-center justify-center mx-auto mb-4">
            <FileText className="h-8 w-8 text-white" />
          </div>
          <h3 className="text-xl font-semibold mb-2">Document not found</h3>
          <p className="text-muted-foreground mb-4">
            The document you're looking for doesn't exist or has been deleted.
          </p>
          <Link href="/dashboard">
            <Button className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700">
              Back to Dashboard
            </Button>
          </Link>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto py-8 px-4">
        <motion.div
          className="mb-8"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            <motion.div 
              className="space-y-2"
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.2 }}
            >
              <div className="flex items-center gap-3">
                <div className="w-12 h-12 bg-gradient-to-r from-blue-600 to-purple-600 rounded-xl flex items-center justify-center shadow-md">
                  <FileUp className="h-6 w-6 text-white" />
                </div>
                <div>
                  <h1 className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                    {document.fileName}
                  </h1>
                  <div className="flex items-center gap-4 text-sm text-muted-foreground mt-1">
                    <div className="flex items-center gap-1">
                      <Calendar className="h-4 w-4" />
                      <span>Uploaded {format(new Date(document.uploadedAt), "MMM dd, yyyy 'at' HH:mm")}</span>
                    </div>
                    <Badge variant="secondary">AI Ready</Badge>
                  </div>
                </div>
              </div>
            </motion.div>

            <motion.div 
              className="flex items-center gap-3"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: 0.3 }}
            >
              <Button
                variant="outline"
                size="sm"
                onClick={async () => {
                  let shareUrl = fileUrl;
                  // Ensure absolute URL
                  if (shareUrl && !/^https?:\/\//i.test(shareUrl)) {
                    shareUrl = window.location.origin + (shareUrl.startsWith("/") ? shareUrl : "/" + shareUrl);
                  }
                  if (navigator.share) {
                    try {
                      await navigator.share({
                        title: document.fileName,
                        url: shareUrl,
                      });
                    } catch (err) {
                      // Fallback: copy to clipboard and show a toast
                      await navigator.clipboard.writeText(shareUrl);
                      toast.toast({ title: "Link copied to clipboard!" });
                    }
                  } else {
                    await navigator.clipboard.writeText(shareUrl);
                    toast.toast({ title: "Link copied to clipboard!" });
                  }
                }}
              >
                <Share2 className="h-4 w-4" />
                Share
              </Button>
              <Button 
                variant="destructive" 
                size="sm" 
                onClick={handleDelete} 
                className="gap-2 hover:shadow-md transition-shadow"
              >
                <Trash2 className="h-4 w-4" />
                Delete
              </Button>
            </motion.div>
          </div>
        </motion.div>

        <div className="grid lg:grid-cols-2 gap-8">
          <motion.div 
            variants={fadeInUp}
            initial="initial"
            animate="animate"
          >
            <Card className="h-full bg-gradient-to-br from-background to-muted/30 border-0 shadow-xl">
              <CardHeader className="border-b border-border/40">
                <CardTitle className="flex items-center gap-3">
                  <div className="w-8 h-8 bg-gradient-to-r from-green-500 to-emerald-500 rounded-lg flex items-center justify-center shadow-md">
                    <FileUp className="h-4 w-4 text-white" />
                  </div>
                  Document Preview
                </CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <div className="h-96 overflow-auto">
                  {!fileUrl || fileUrl === backendBaseUrl ? (
                    <div className="flex items-center justify-center h-full text-muted-foreground">
                      <div className="text-center">
                        <FileText className="h-12 w-12 mx-auto mb-4 opacity-50" />
                        <p>No preview available</p>
                      </div>
                    </div>
                  ) : isPdf ? (
                    <iframe
                      src={fileUrl}
                      width="100%"
                      height="100%"
                      style={{ minHeight: "24rem", border: "none" }}
                      title="PDF Preview"
                      className="rounded-b-lg"
                    />
                  ) : previewContent !== null ? (
                    <div className="p-6">
                      <pre className="whitespace-pre-wrap text-sm leading-relaxed font-mono bg-muted/50 p-4 rounded-lg">
                        {previewContent}
                      </pre>
                    </div>
                  ) : (
                    <div className="flex items-center justify-center h-full">
                      <div className="text-center">
                        <div className="w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-4"></div>
                        <p className="text-muted-foreground">Loading preview...</p>
                      </div>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </motion.div>

          <motion.div 
            variants={fadeInUp}
            initial="initial"
            animate="animate"
            transition={{ delay: 0.2 }}
          >
            <DocumentChat documentId={id as string} />
          </motion.div>
        </div>
      </div>
    </div>
  )
}