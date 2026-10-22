import { useDocumentTitle } from '../lib/useDocumentTitle'

type PermissionDeniedProps = {
  permission: string
  title?: string
}

export function PermissionDenied({
  permission,
  title = 'Access denied',
}: PermissionDeniedProps) {
  useDocumentTitle('Access denied')

  return (
    <div className="stack">
      <header>
        <p className="section-label">Security</p>
        <h1 className="title">{title}</h1>
        <p className="lede">
          Missing permission for this page.
        </p>
      </header>
      <p className="error" role="alert">
        Missing permission: {permission}
      </p>
    </div>
  )
}
