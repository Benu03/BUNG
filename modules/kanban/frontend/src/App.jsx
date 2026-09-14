import { useState } from 'react'
import BoardList from './components/BoardList.jsx'
import Board from './components/Board.jsx'

export default function App() {
  const [boardId, setBoardId] = useState(null)

  return (
    <div className="app">
      <header>
        <div>
          <h1>Kanban</h1>
          <p>Boards, columns and cards.</p>
        </div>
      </header>

      {boardId
        ? <Board boardId={boardId} onBack={() => setBoardId(null)} />
        : <BoardList onOpen={setBoardId} />}
    </div>
  )
}
