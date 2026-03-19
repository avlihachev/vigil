export type ActionType = 'edit' | 'read' | 'bash' | 'search' | 'waiting' | 'confirm';

export interface Action {
  type: ActionType;
  target: string;
  result?: string;
  timestamp: number;
}

export type SessionStatus = 'active' | 'waiting' | 'confirm' | 'idle';

export interface Session {
  pid: number;
  sessionId: string;
  cwd: string;
  startedAt: number;
  source: string;
  projectName: string;
  name: string;
  status: SessionStatus;
  duration: string;
  tokensIn: string;
  tokensOut: string;
  recentActions: Action[];
  sibling?: string;
}

export interface HistoricalSession {
  sessionId: string;
  name: string;
  lastActiveAt: number;
  tokensIn: string;
  tokensOut: string;
}

export interface ProjectHistory {
  projectName: string;
  cwd: string;
  sessions: HistoricalSession[];
}

export interface Settings {
  notifyConfirm: boolean;
  notifyWaiting: boolean;
  badgeConfirm: boolean;
  badgeWaiting: boolean;
  badgeActive: boolean;
}
