export default function handler(req, res) {
  res.status(200).json({ 
    message: 'Hello from Next.js API route',
    timestamp: new Date().toISOString(),
    method: req.method
  });
}