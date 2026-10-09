export interface Identity {
  uuid: string;
  nickname: string;
  avatar: string;
}
export interface DayActivity {
  date: string;
  users: number;
  messages: number;
  posts: number;
}
export interface Overview {
  users: number;
  groups: number;
  posts: number;
  todayMessages: number;
  todayUsers: number;
  knowledge: number;
  activity: DayActivity[];
  generatedAt: string;
}
export interface System {
  uptimeSeconds: number;
  heapBytes: number;
  goroutines: number;
  onlineUsers: number;
  services: { name: string; status: "up" | "down"; detail: string }[];
  grafanaUrl: string;
  kibanaUrl: string;
  generatedAt: string;
}
