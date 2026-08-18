/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { api } from '@/lib/api'

import type { PayloadLogDetail, PayloadLogListData } from './types'

export interface GetPayloadLogsParams {
  page?: number
  page_size?: number
  username?: string
  model_name?: string
  request_id?: string
}

export async function getPayloadLogs(params: GetPayloadLogsParams = {}) {
  const res = await api.get<{ data: PayloadLogListData }>('/api/payload_log/', {
    params,
  })
  return res.data?.data
}

export async function getPayloadLogDetail(id: number) {
  const res = await api.get<{ data: PayloadLogDetail }>(
    `/api/payload_log/${id}`
  )
  return res.data?.data
}

// The platform-wide switch is stored as the PayloadLogEnabled option
// (root-only, read/written through the generic option endpoints).
export async function getPayloadLogEnabled() {
  const res = await api.get<{ data: { key: string; value: string }[] }>(
    '/api/option/'
  )
  const opt = res.data?.data?.find((o) => o.key === 'PayloadLogEnabled')
  return opt?.value === 'true'
}

export async function setPayloadLogEnabled(enabled: boolean) {
  const res = await api.put('/api/option/', {
    key: 'PayloadLogEnabled',
    value: enabled,
  })
  return res.data
}
