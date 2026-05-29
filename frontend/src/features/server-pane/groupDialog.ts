// Discriminated payload accepted by the Group dialog (`DialogType.Group`).
//
// - `string`: groupId, opens the dialog in edit/rename mode.
// - object: opens the dialog in create mode, optionally prefilled with a
//   parent group and optionally with a callback invoked with the new id.
export type GroupDialogData =
  | string
  | {
      parentId?: string
      onCreated?: (id: string) => void
    }

export function isEditPayload(data: unknown): data is string {
  return typeof data === 'string' && data.length > 0
}

export function asCreatePayload(
  data: unknown,
): { parentId?: string; onCreated?: (id: string) => void } | undefined {
  if (data == null || typeof data !== 'object') {
    return undefined
  }
  return data as { parentId?: string; onCreated?: (id: string) => void }
}
