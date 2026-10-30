export type NotificationStatus = 'pending' | 'processing' | 'done' | 'failed';

export interface AdminSession {
  email: string;
}

export interface NotificationSummary {
  id: number;
  title: string;
  status: NotificationStatus;
  sent: number;
  total: number;
  failed: number;
  created_at: string;
}

export interface NotificationList {
  items: NotificationSummary[];
  total: number;
}

export interface NotificationJob {
  status: NotificationStatus;
  high_watermark: number;
  cursor: number;
  total: number;
  sent: number;
  failed: number;
  invalid: number;
  attempts: number;
  next_attempt_at: string;
  lease_until?: string;
  last_error?: string;
  started_at?: string;
  completed_at?: string;
}

export interface NotificationDetails {
  id: number;
  title: string;
  body: string;
  created_by: number;
  created_at: string;
  job: NotificationJob;
}

export interface NotificationBatch {
  notification_id: number;
  batch_no: number;
  token_count: number;
  success: number;
  failure: number;
  invalid: number;
  status: 'sending' | 'done' | 'failed';
  error?: string;
  started_at: string;
  completed_at?: string;
}

export interface BatchList {
  items: NotificationBatch[];
  next_before: number | null;
}

export interface NotificationStats {
  total: number;
  pending: number;
  processing: number;
  done: number;
  failed: number;
}
