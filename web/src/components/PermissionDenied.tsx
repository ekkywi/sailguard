type PermissionDeniedProps = {
  permission: string
  title?: string
}

export function PermissionDenied({
  permission,
  title = 'Access denied',
}: PermissionDeniedProps) {
  return (
    <div className="stack">
      <header>
        <p className="section-label">Security</p>
        <h1 className="title">{title}</h1>
        <p className="lede">
          Your account does not include the permission required for this page.
        </p>
      </header>
      <p className="error" role="alert">
        Missing permission: {permission}
      </p>
    </div>
  )
}
