"use client"
import { useQuery } from "@tanstack/react-query"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"
import Link from "next/link"
import { FileUp, Plus, Calendar, TrendingUp, FileText, MessageSquare } from "lucide-react"
import api from "@/lib/api"
import { useAuthContext } from "@/context/AuthContext"
import { redirect } from "next/navigation"
import { format } from "date-fns"
import { getAuth } from "firebase/auth"
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

const staggerContainer = {
  initial: {},
  animate: {
    transition: {
      staggerChildren: 0.1,
      delayChildren: 0.2
    }
  }
}

export default function DashboardPage() {
  const { user } = useAuthContext()

  if (!user) {
    redirect("/login")
  }

  const auth = getAuth()
  const { data: documents, isLoading } = useQuery<Document[]>({
    queryKey: ["documents"],
    queryFn: async () => {
      const user = auth.currentUser
      if (!user) throw new Error("Not authenticated")
      const token = await user.getIdToken()
      const { data } = await api.get("/api/documents", {
        headers: { Authorization: `Bearer ${token}` },
      })
      return data
    },
  })

  return (
    <div className="container mx-auto py-8 px-4 max-h-[80vh] overflow-y-auto">
      {/* Header Section */}
      <motion.div
        className="mb-8"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6 }}
      >
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <motion.h1 
              className="text-3xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.2 }}
            >
              Welcome back, {user.email?.split("@")[0]}
            </motion.h1>
            <motion.p 
              className="text-muted-foreground mt-2"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.3 }}
            >
              Here's what's happening with your documents today
            </motion.p>
          </div>
          <motion.div
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.4 }}
          >
            <Link href="/upload">
              <Button
                size="lg"
                className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 shadow-lg hover:shadow-xl transition-all duration-300 group"
              >
                <Plus className="mr-2 h-5 w-5 group-hover:rotate-90 transition-transform" />
                Upload Document
              </Button>
            </Link>
          </motion.div>
        </div>
        <motion.div
          className="mb-2"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.35 }}
        >
          <p className="text-xs text-blue-700 dark:text-blue-300 italic">Prefer to upload in text format for best results.</p>
        </motion.div>
      </motion.div>

      {/* Stats Cards */}
      <motion.div
        className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8"
        variants={staggerContainer}
        initial="initial"
        animate="animate"
      >
        <motion.div variants={fadeInUp}>
          <Card className="bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950/50 dark:to-blue-900/50 border border-blue-200 dark:border-blue-800 hover:shadow-lg transition-all">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-blue-600 dark:text-blue-400 mb-1">Total Documents</p>
                  <p className="text-3xl font-bold text-blue-700 dark:text-blue-300">{documents?.length || 0}</p>
                  <p className="text-xs text-blue-500 dark:text-blue-400 mt-2">+2 from last week</p>
                </div>
                <div className="h-12 w-12 bg-blue-600 rounded-xl flex items-center justify-center shadow-md">
                  <FileText className="h-6 w-6 text-white" />
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div variants={fadeInUp}>
          <Card className="bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950/50 dark:to-purple-900/50 border border-purple-200 dark:border-purple-800 hover:shadow-lg transition-all">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-purple-600 dark:text-purple-400 mb-1">AI Insights</p>
                  <p className="text-3xl font-bold text-purple-700 dark:text-purple-300">
                    {documents?.length ? documents.length * 3 : 0}
                  </p>
                  <p className="text-xs text-purple-500 dark:text-purple-400 mt-2">+5 from last week</p>
                </div>
                <div className="h-12 w-12 bg-purple-600 rounded-xl flex items-center justify-center shadow-md">
                  <TrendingUp className="h-6 w-6 text-white" />
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>

        <motion.div variants={fadeInUp}>
          <Card className="bg-gradient-to-br from-green-50 to-green-100 dark:from-green-950/50 dark:to-green-900/50 border border-green-200 dark:border-green-800 hover:shadow-lg transition-all">
            <CardContent className="p-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-green-600 dark:text-green-400 mb-1">Chat Sessions</p>
                  <p className="text-3xl font-bold text-green-700 dark:text-green-300">
                    {documents?.length ? documents.length * 2 : 0}
                  </p>
                  <p className="text-xs text-green-500 dark:text-green-400 mt-2">+3 from last week</p>
                </div>
                <div className="h-12 w-12 bg-green-600 rounded-xl flex items-center justify-center shadow-md">
                  <MessageSquare className="h-6 w-6 text-white" />
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      </motion.div>

      {/* Documents Section */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.6, delay: 0.4 }}
      >
        <div className="flex items-center justify-between mb-6">
          <motion.h2 
            className="text-2xl font-bold"
            initial={{ opacity: 0, x: -10 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.5 }}
          >
            Your Documents
          </motion.h2>
          <Badge variant="secondary" className="px-3 py-1 bg-muted/50 backdrop-blur-sm">
            {documents?.length || 0} documents
          </Badge>
        </div>

        {isLoading ? (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {[...Array(3)].map((_, i) => (
              <Card key={i} className="h-40">
                <CardContent className="p-6">
                  <Skeleton className="h-4 w-3/4 mb-4" />
                  <Skeleton className="h-3 w-1/2 mb-2" />
                  <Skeleton className="h-3 w-2/3" />
                </CardContent>
              </Card>
            ))}
          </div>
        ) : documents && documents.length > 0 ? (
          <motion.div
            className="grid gap-6 md:grid-cols-2 lg:grid-cols-3"
            variants={staggerContainer}
            initial="initial"
            animate="animate"
          >
            {documents.map((doc, index) => (
              <motion.div 
                key={doc.id} 
                variants={fadeInUp}
                whileHover={{ y: -5 }}
                transition={{ duration: 0.2 }}
              >
                <Link href={`/documents/${doc.id}`}>
                  <Card className="h-full group hover:shadow-xl transition-all duration-300 border-0 bg-gradient-to-br from-background to-muted/30 hover:from-background hover:to-muted/50">
                    <CardHeader className="pb-3">
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-3">
                          <div className="rounded-lg bg-gradient-to-r from-blue-600 to-purple-600 p-3 group-hover:scale-110 transition-transform duration-300 shadow-md">
                            <FileUp className="h-5 w-5 text-white" />
                          </div>
                          <div className="flex-1 min-w-0">
                            <CardTitle className="text-lg font-semibold truncate group-hover:text-blue-600 transition-colors">
                              {doc.fileName}
                            </CardTitle>
                          </div>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <div className="flex items-center gap-2 text-sm text-muted-foreground">
                        <Calendar className="h-4 w-4" />
                        <span>{format(new Date(doc.uploadedAt), "MMM dd, yyyy")}</span>
                      </div>
                      <div className="mt-4 flex gap-2">
                        <Badge variant="secondary" className="text-xs">
                          AI Ready
                        </Badge>
                        <Badge variant="outline" className="text-xs">
                          Analyzed
                        </Badge>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              </motion.div>
            ))}
          </motion.div>
        ) : (
          <motion.div
            initial={{ opacity: 0, scale: 0.95 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.5 }}
          >
            <Card className="p-12 text-center bg-gradient-to-br from-muted/30 to-muted/60 border-dashed border-2">
              <div className="mx-auto max-w-md">
                <div className="mx-auto w-16 h-16 bg-gradient-to-r from-blue-600 to-purple-600 rounded-2xl flex items-center justify-center mb-6 shadow-lg">
                  <FileUp className="h-8 w-8 text-white" />
                </div>
                <h3 className="text-xl font-semibold mb-3">No documents yet</h3>
                <p className="text-muted-foreground mb-6 leading-relaxed">
                  Upload your first document to get started with AI-powered analysis and strategic insights
                </p>
                <Link href="/upload">
                  <Button
                    size="lg"
                    className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 shadow-lg"
                  >
                    <Plus className="mr-2 h-5 w-5" />
                    Upload Your First Document
                  </Button>
                </Link>
              </div>
            </Card>
          </motion.div>
        )}
      </motion.div>
    </div>
  )
}