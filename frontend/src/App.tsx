import { Routes, Route, Navigate } from 'react-router-dom'
import { Layout } from '@/components/layout/Layout'
import { ProtectedRoute, AdminRoute } from '@/components/common/ProtectedRoute'
import { ProblemsPage } from '@/pages/ProblemsPage'
import { ProblemPage } from '@/pages/ProblemPage'
import { ProblemSubmissionsPage } from '@/pages/ProblemSubmissionsPage'
import { SubmissionDetailPage } from '@/pages/SubmissionDetailPage'
import { UserProfilePage } from '@/pages/UserProfilePage'
import { NoticesPage } from '@/pages/NoticesPage'
import { LoginPage } from '@/pages/LoginPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { useAuth } from '@/hooks/useAuth'
import { HelpPage } from '@/pages/HelpPage'
import { ContactPage } from '@/pages/ContactPage'

// Admin pages
import { AdminDashboard } from '@/pages/admin/AdminDashboard'
import { AdminNotices } from '@/pages/admin/AdminNotices'
import { AdminProblemsUpload } from '@/pages/admin/AdminProblemsUpload'
import { AdminProblemsVisibility } from '@/pages/admin/AdminProblemsVisibility'
import { AdminUsersBulk } from '@/pages/admin/AdminUsersBulk'
import { AdminSubmissionTest } from '@/pages/admin/AdminSubmissionTest'
import { AdminSystem } from '@/pages/admin/AdminSystem'
import { AdminUsersList } from '@/pages/admin/AdminUsersList'

function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<RootRedirect />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/help" element={<HelpPage />} />
        <Route path="/contact" element={<ContactPage />} />
        <Route path="*" element={<NotFoundPage />} />
        
        {/* 認証が必要なルート */}
        <Route element={<ProtectedRoute />}>
          <Route path="/problems" element={<ProblemsPage />} />
          <Route path="/problems/:id" element={<ProblemPage />} />
          <Route path="/problems/:id/submissions" element={<ProblemSubmissionsPage />} />
          <Route path="/submissions/:id" element={<SubmissionDetailPage />} />
          <Route path="/users/:userid" element={<UserProfilePage />} />
          <Route path="/notices" element={<NoticesPage />} />
        </Route>
        
        {/* 管理者専用ルート */}
        <Route element={<AdminRoute />}>
          <Route path="/admin" element={<AdminDashboard />} />
          <Route path="/admin/notices" element={<AdminNotices />} />
          <Route path="/admin/problems/upload" element={<AdminProblemsUpload />} />
          <Route path="/admin/problems/visibility" element={<AdminProblemsVisibility />} />
          <Route path="/admin/users/bulk" element={<AdminUsersBulk />} />
          <Route path="/admin/submissions/test" element={<AdminSubmissionTest />} />
          <Route path="/admin/system" element={<AdminSystem />} />
          <Route path="/admin/users" element={<AdminUsersList />} />
        </Route>
      </Route>
    </Routes>
  )
}

function RootRedirect() {
  const { user, isLoading } = useAuth()

  if (isLoading) {
    return null
  }

  return <Navigate to={user ? '/problems' : '/login'} replace />
}

export default App
