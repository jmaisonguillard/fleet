export default function Home() {
  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui' }}>
      <h1>Next.js Test Application</h1>
      <p>Running with Fleet Node.js runtime</p>
      <div style={{ marginTop: '2rem' }}>
        <h2>Environment Info:</h2>
        <ul>
          <li>API URL: {process.env.NEXT_PUBLIC_API_URL || 'not configured'}</li>
          <li>Node Version: {process.version}</li>
        </ul>
      </div>
    </div>
  );
}