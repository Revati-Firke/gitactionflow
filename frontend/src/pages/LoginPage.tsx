import { loginURL } from '../services/api'

type LoginPageProps = {
  error?: string | null
}

export function LoginPage({ error }: LoginPageProps) {
  return (
    <div className="login">
      <p className="eyebrow">Event-driven Git automation</p>
      <h1 className="login-brand">GitActionFlow</h1>
      <p className="lede">
        Sign in with GitHub to connect one repository, configure rules, and review webhook activity.
      </p>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      <a className="btn lg" href={loginURL()}>
        Continue with GitHub
      </a>
      <p className="note">Authentication uses a secure HttpOnly session cookie. Tokens never reach the browser.</p>
    </div>
  )
}
