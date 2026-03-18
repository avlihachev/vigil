export type ActionType = 'edit' | 'read' | 'bash' | 'search' | 'waiting';

export interface Action {
  type: ActionType;
  target: string;
  result?: string;
  timestamp: number;
}

export type SessionStatus = 'active' | 'waiting' | 'idle';

export interface Session {
  pid: number;
  sessionId: string;
  cwd: string;
  startedAt: number;
  source: string;
  projectName: string;
  status: SessionStatus;
  duration: string;
  recentActions: Action[];
}
