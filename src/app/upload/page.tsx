"use client"

import type React from "react"

import { useRouter } from "next/navigation"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Button } from "@/components/ui/button"
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { FileUp, Upload, CheckCircle, X, ArrowLeft, Sparkles } from "lucide-react"
import { useToast } from "@/components/ui/toaster"
import api from "@/lib/api"
import { getAuth } from "firebase/auth"
import { useState } from "react"
import { motion } from "framer-motion"
import Link from "next/link"


const formSchema = z.object({
  file: z.any().refine((file) => file instanceof FileList && file.length === 1, "File is required"),
})

const fadeInUp = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  transition: { duration: 0.5 },
}

export default function UploadPage() {
  const router = useRouter()
  const toast = useToast()
  const [dragActive, setDragActive] = useState(false)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [isUploading, setIsUploading] = useState(false)

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  })

  const fileRef = form.register("file")

  const handleDrag = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (e.type === "dragenter" || e.type === "dragover") {
      setDragActive(true)
    } else if (e.type === "dragleave") {
      setDragActive(false)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setDragActive(false)

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const file = e.dataTransfer.files[0]
      setSelectedFile(file)
      const dt = new DataTransfer()
      dt.items.add(file)
      form.setValue("file", dt.files)
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setSelectedFile(e.target.files[0])
    }
  }

  const removeFile = () => {
    setSelectedFile(null)
    form.setValue("file", new DataTransfer().files)
  }

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setIsUploading(true)
    try {
      const auth = getAuth()
      const user = auth.currentUser
      if (!user) {
        toast.toast({ title: "You must be logged in to upload documents" })
        return
      }
      const token = await user.getIdToken(true)

      const formData = new FormData()
      formData.append("document", values.file[0])

      const { data } = await api.post("/api/documents", formData, {
        headers: {
          "Content-Type": "multipart/form-data",
          Authorization: `Bearer ${token}`,
        },
      })

      toast.toast({ title: "Document uploaded successfully! Redirecting to analysis..." })
      router.push(`/documents/${data.id}`)
    } catch (error) {
      toast.toast({ title: "Upload failed: " + (error instanceof Error ? error.message : "Unknown error") })
    } finally {
      setIsUploading(false)
    }
  }

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto py-8 px-4 max-w-4xl">
        <motion.div
          className="mb-8"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
        >
          <div className="flex items-center gap-4 mb-6">
            <Link href="/dashboard">
              <Button variant="ghost" size="sm" className="gap-2">
                <ArrowLeft className="h-4 w-4" />
                Back to Dashboard
              </Button>
            </Link>
          </div>

          <div className="text-center">
            <motion.div
              className="mx-auto w-16 h-16 bg-gradient-to-br from-blue-600 to-purple-600 rounded-2xl flex items-center justify-center mb-4 shadow-lg"
              whileHover={{ scale: 1.05 }}
              transition={{ type: "spring", stiffness: 300 }}
            >
              <FileUp className="h-8 w-8 text-white" />
            </motion.div>
            <h1 className="text-4xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent mb-2">
              Upload Document
            </h1>
            <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
              Upload your document to start analyzing with AI-powered insights and strategic recommendations
            </p>
            <div className="flex items-center justify-center gap-2 mt-4">
              <Sparkles className="w-4 h-4 text-blue-600" />
              <span className="text-sm text-muted-foreground">AI analysis ready in seconds</span>
            </div>
          </div>
        </motion.div>

        <motion.div variants={fadeInUp} initial="initial" animate="animate">
          <Card className="border-0 shadow-2xl bg-gradient-to-br from-background to-muted/30">
            <CardHeader className="border-b border-border/40">
              <CardTitle className="flex items-center gap-3 text-2xl">
                <div className="w-8 h-8 bg-gradient-to-r from-green-500 to-emerald-500 rounded-lg flex items-center justify-center">
                  <Upload className="h-4 w-4 text-white" />
                </div>
                Document Upload
              </CardTitle>
              <CardDescription className="text-base">
                Supported formats: PDF, TXT files up to 10MB. Your documents are processed securely and privately.
              </CardDescription>
            </CardHeader>
            <CardContent className="p-8">
              <Form {...form}>
                <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
                  <FormField
                    control={form.control}
                    name="file"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel className="text-lg font-semibold">Select Document</FormLabel>
                        <FormControl>
                          <div
                            className={`relative border-2 border-dashed rounded-xl p-12 text-center transition-all duration-300 ${
                              dragActive
                                ? "border-blue-500 bg-blue-50 dark:bg-blue-950/20 scale-[1.02]"
                                : selectedFile
                                  ? "border-green-500 bg-green-50 dark:bg-green-950/20"
                                  : "border-muted-foreground/25 hover:border-blue-400 hover:bg-muted/30"
                            }`}
                            onDragEnter={handleDrag}
                            onDragLeave={handleDrag}
                            onDragOver={handleDrag}
                            onDrop={handleDrop}
                          >
                            {selectedFile ? (
                              <motion.div
                                className="space-y-4"
                                initial={{ opacity: 0, scale: 0.9 }}
                                animate={{ opacity: 1, scale: 1 }}
                                transition={{ duration: 0.3 }}
                              >
                                <div className="flex items-center justify-center gap-4">
                                  <div className="rounded-full bg-green-100 p-3 dark:bg-green-900/20">
                                    <CheckCircle className="h-8 w-8 text-green-600" />
                                  </div>
                                  <div className="text-left">
                                    <p className="font-semibold text-lg">{selectedFile.name}</p>
                                    <p className="text-muted-foreground">
                                      {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                                    </p>
                                    <Badge variant="secondary" className="mt-1">
                                      {selectedFile.type || "Unknown type"}
                                    </Badge>
                                  </div>
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={removeFile}
                                    className="ml-auto hover:bg-red-100 hover:text-red-600"
                                  >
                                    <X className="h-4 w-4" />
                                  </Button>
                                </div>
                              </motion.div>
                            ) : (
                              <div className="space-y-6">
                                <motion.div
                                  className="mx-auto w-20 h-20 bg-gradient-to-r from-blue-500 to-purple-500 rounded-2xl flex items-center justify-center"
                                  whileHover={{ scale: 1.05 }}
                                  transition={{ type: "spring", stiffness: 300 }}
                                >
                                  <Upload className="h-10 w-10 text-white" />
                                </motion.div>
                                <div>
                                  <p className="text-xl font-semibold mb-2">
                                    Drop your file here, or{" "}
                                    <span className="text-blue-600 hover:text-blue-700 cursor-pointer">browse</span>
                                  </p>
                                  <p className="text-muted-foreground">
                                    PDF, TXT files up to 10MB • Drag and drop or click to select
                                  </p>
                                </div>
                                <div className="flex items-center justify-center gap-6 text-sm text-muted-foreground">
                                  <div className="flex items-center gap-2">
                                    <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                                    <span>Secure upload</span>
                                  </div>
                                  <div className="flex items-center gap-2">
                                    <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                                    <span>AI ready</span>
                                  </div>
                                  <div className="flex items-center gap-2">
                                    <div className="w-2 h-2 bg-purple-500 rounded-full"></div>
                                    <span>Fast processing</span>
                                  </div>
                                </div>
                              </div>
                            )}
                            <Input
                              type="file"
                              accept=".pdf,.txt"
                              className="absolute inset-0 w-full h-full opacity-0 cursor-pointer"
                              {...fileRef}
                              onChange={(e) => {
                                field.onChange(e.target.files)
                                handleFileChange(e)
                              }}
                            />
                          </div>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <div className="flex justify-between items-center pt-6 border-t border-border/40">
                    <div className="text-sm text-muted-foreground">
                      <p>Your document will be processed securely and analyzed with AI</p>
                    </div>
                    <div className="flex gap-4">
                      <Button type="button" variant="outline" onClick={() => router.back()} disabled={isUploading}>
                        Cancel
                      </Button>
                      <Button
                        type="submit"
                        disabled={!selectedFile || isUploading}
                        className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 px-8 shadow-lg"
                      >
                        {isUploading ? (
                          <div className="flex items-center gap-2">
                            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div>
                            Uploading...
                          </div>
                        ) : (
                          <>
                            <Upload className="h-4 w-4 mr-2" />
                            Upload & Analyze
                          </>
                        )}
                      </Button>
                    </div>
                  </div>
                </form>
              </Form>
            </CardContent>
          </Card>
        </motion.div>

        {/* Features Preview */}
        <motion.div
          className="mt-12 grid md:grid-cols-3 gap-6"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.3 }}
        >
          {[
            {
              icon: "🧠",
              title: "AI Analysis",
              description: "Advanced AI extracts key insights and strategic recommendations",
            },
            {
              icon: "💬",
              title: "Interactive Chat",
              description: "Ask questions about your document and get instant answers",
            },
            {
              icon: "📊",
              title: "Visual Insights",
              description: "Get charts, summaries, and actionable business intelligence",
            },
          ].map((feature, index) => (
            <Card key={index} className="text-center p-6 bg-gradient-to-br from-background to-muted/20 border-0">
              <div className="text-3xl mb-3">{feature.icon}</div>
              <h3 className="font-semibold mb-2">{feature.title}</h3>
              <p className="text-sm text-muted-foreground">{feature.description}</p>
            </Card>
          ))}
        </motion.div>
      </div>
    </div>
  )
}
