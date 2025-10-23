import { BrowserRouter as Router, Routes, Route } from 'react-router-dom'

function App() {
  return (
    <Router>
      <div className="min-h-screen bg-gray-100">
        <Routes>
          <Route path="/" element={<h1 className="text-4xl p-8">P2P Lending Platform</h1>} />
          <Route path="/lending" element={<h1 className="text-4xl p-8">Lending</h1>} />
          <Route path="/borrowing" element={<h1 className="text-4xl p-8">Borrowing</h1>} />
          <Route path="/credit-score" element={<h1 className="text-4xl p-8">Credit Score</h1>} />
        </Routes>
      </div>
    </Router>
  )
}

export default App
