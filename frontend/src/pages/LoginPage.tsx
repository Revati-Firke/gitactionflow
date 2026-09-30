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
        Connect a repository, define rules, and turn issues and pull requests into GitHub actions and Slack alerts.
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
