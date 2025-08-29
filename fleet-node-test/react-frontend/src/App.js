import React, { useState, useEffect } from 'react';

function App() {
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Try to fetch from Express API if available
    const apiUrl = process.env.REACT_APP_API_URL || 'http://localhost:3000';
    fetch(`${apiUrl}/api/users`)
      .then(res => res.json())
      .then(data => {
        setData(data);
        setLoading(false);
      })
      .catch(err => {
        console.error('API fetch error:', err);
        setLoading(false);
      });
  }, []);

  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui' }}>
      <h1>React Frontend Test</h1>
      <p>Running with Fleet - Build Mode Example</p>
      
      <div style={{ marginTop: '2rem' }}>
        <h2>Build Info:</h2>
        <ul>
          <li>API URL: {process.env.REACT_APP_API_URL || 'not configured'}</li>
          <li>Environment: {process.env.NODE_ENV}</li>
        </ul>
      </div>

      <div style={{ marginTop: '2rem' }}>
        <h2>API Data:</h2>
        {loading ? (
          <p>Loading...</p>
        ) : data ? (
          <pre>{JSON.stringify(data, null, 2)}</pre>
        ) : (
          <p>No data available</p>
        )}
      </div>
    </div>
  );
}

export default App;