import { Link, Route, Routes } from 'react-router-dom'

function Home() {
  return (
    <main className="page">
      <p className="eyebrow">SailGuard</p>
      <h1>Control plane</h1>
      <p className="lede">
        Admin UI scaffold. Auth, devices, and policies arrive in later milestones.
      </p>
      <p>
        <a href="/v1/health" target="_blank" rel="noreferrer">
          Check API /v1/health
        </a>
      </p>
    </main>
  )
}

export default function App() {
  return (
    <div className="shell">
      <header className="top">
        <Link to="/" className="brand">
          SailGuard
        </Link>
        <nav>
          <span className="muted">MVP scaffold</span>
        </nav>
      </header>
      <Routes>
        <Route path="/" element={<Home />} />
      </Routes>
    </div>
  )
}
