// client-template.ts — Add this section to frontend/src/api/client.ts
// Replace "Foo"/"foo" with your entity name throughout.

// --- Foo ---
// TypeScript interface mirror of the Go models.Foo struct.
// Field names must exactly match the JSON tags in models.go (snake_case).
export interface Foo {
  id: number;           // gorm.Model.ID  (Go uint → TS number)
  group_id: number;
  name: string;
  description: string;
  created_at: string;   // ISO 8601 timestamp
  updated_at: string;
  deleted_at?: string | null;
}

// Input type for create/update operations (all fields optional).
export type FooInput = Pick<Foo, 'name' | 'description'>;

// Centralized API methods — call these from components/hooks, never call axios directly.
export const fooApi = {
  /** GET /groups/:groupId/foos — list all foos for a group */
  getAll: (groupId: number) =>
    api.get<Foo[]>(`/groups/${groupId}/foos`),

  /** GET /groups/:groupId/foos/:id — get a single foo */
  getById: (groupId: number, id: number) =>
    api.get<Foo>(`/groups/${groupId}/foos/${id}`),

  /** POST /groups/:groupId/foos — create a new foo */
  create: (groupId: number, data: FooInput) =>
    api.post<Foo>(`/groups/${groupId}/foos`, data),

  /** PUT /groups/:groupId/foos/:id — update an existing foo */
  update: (groupId: number, id: number, data: Partial<FooInput>) =>
    api.put<Foo>(`/groups/${groupId}/foos/${id}`, data),

  /** DELETE /groups/:groupId/foos/:id — soft-delete a foo */
  delete: (groupId: number, id: number) =>
    api.delete(`/groups/${groupId}/foos/${id}`),
};
