import { Outlet } from 'react-router-dom'
import { Header } from './Header'
import { Link } from 'react-router-dom'

export function Layout() {
  return (
    <div className="min-h-screen flex flex-col">
      <Header />
      <main className="flex-1">
        <div className="app-container">
          <Outlet />
        </div>
      </main>
      <footer className="footer">
        <div className="app-container">
          <div className="flex flex-col sm:flex-row justify-between items-center gap-4 text-sm">
            <p className="text-muted">© 2025 TUIS Online Judge</p>
            <nav aria-label="フッターナビゲーション" className="flex items-center gap-3">
              <Link
                to="/help"
                className="text-muted hover:text-primary transition-colors hover:underline"
              >
                ヘルプ
              </Link>
              <span className="text-muted">|</span>
              <Link
                to="/contact"
                className="text-muted hover:text-primary transition-colors hover:underline"
              >
                お問い合わせ
              </Link>
            </nav>
          </div>
        </div>
      </footer>
    </div>
  )
}
