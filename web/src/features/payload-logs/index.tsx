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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import {
  getPayloadLogDetail,
  getPayloadLogEnabled,
  getPayloadLogs,
  setPayloadLogEnabled,
} from './api'

const PAGE_SIZE = 20

function formatTime(seconds: number) {
  if (!seconds) return '-'
  const d = new Date(seconds * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export function PayloadLogs() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [detailId, setDetailId] = useState<number | null>(null)

  const { data: enabled } = useQuery({
    queryKey: ['payload-log-enabled'],
    queryFn: getPayloadLogEnabled,
  })

  const { data, isLoading } = useQuery({
    queryKey: ['payload-logs', page],
    queryFn: () => getPayloadLogs({ page, page_size: PAGE_SIZE }),
  })

  const { data: detail, isLoading: detailLoading } = useQuery({
    queryKey: ['payload-log', detailId],
    queryFn: () => getPayloadLogDetail(detailId as number),
    enabled: detailId != null,
  })

  const onToggle = async (next: boolean) => {
    await setPayloadLogEnabled(next)
    queryClient.invalidateQueries({ queryKey: ['payload-log-enabled'] })
  }

  const items = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Payload Logs')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
          <div className='bg-card rounded-lg border p-4'>
            <div className='flex items-start justify-between gap-4'>
              <div className='space-y-1'>
                <div className='text-sm font-medium'>
                  {t('Business payload logging')}
                </div>
                <p className='text-muted-foreground max-w-2xl text-xs'>
                  {enabled
                    ? t(
                        'Enabling stores the full request and response of every call ensure this complies with your customer agreements'
                      )
                    : t(
                        'Off by default the platform stores only billing metadata never your prompts or responses'
                      )}
                </p>
              </div>
              <Switch checked={!!enabled} onCheckedChange={onToggle} />
            </div>
          </div>

          <div className='rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Time')}</TableHead>
                  <TableHead>{t('User')}</TableHead>
                  <TableHead>{t('Model')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Request ID')}</TableHead>
                  <TableHead className='text-right'>{t('Action')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {isLoading ? (
                  Array.from({ length: 5 }).map((_, i) => (
                    <TableRow key={i}>
                      <TableCell colSpan={6}>
                        <Skeleton className='h-5 w-full' />
                      </TableCell>
                    </TableRow>
                  ))
                ) : items.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className='text-muted-foreground py-10 text-center'
                    >
                      {t('No payload logs yet')}
                    </TableCell>
                  </TableRow>
                ) : (
                  items.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className='whitespace-nowrap'>
                        {formatTime(item.created_at)}
                      </TableCell>
                      <TableCell>{item.username || item.user_id}</TableCell>
                      <TableCell className='font-mono text-xs'>
                        {item.model_name || '-'}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            item.status_code >= 200 && item.status_code < 300
                              ? 'secondary'
                              : 'destructive'
                          }
                        >
                          {item.status_code || '-'}
                        </Badge>
                      </TableCell>
                      <TableCell className='font-mono text-xs'>
                        {item.request_id || '-'}
                      </TableCell>
                      <TableCell className='text-right'>
                        <Button
                          variant='outline'
                          size='sm'
                          onClick={() => setDetailId(item.id)}
                        >
                          {t('View detail')}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>

          <div className='flex items-center justify-between'>
            <span className='text-muted-foreground text-xs'>{total}</span>
            <div className='flex items-center gap-2'>
              <Button
                variant='outline'
                size='sm'
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
              >
                {t('Previous')}
              </Button>
              <span className='text-xs'>
                {page} / {totalPages}
              </span>
              <Button
                variant='outline'
                size='sm'
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                {t('Next')}
              </Button>
            </div>
          </div>

          <Dialog
            open={detailId != null}
            onOpenChange={(open) => !open && setDetailId(null)}
          >
            <DialogContent className='max-w-3xl'>
              <DialogHeader>
                <DialogTitle>{t('Call detail')}</DialogTitle>
              </DialogHeader>
              {detailLoading || !detail ? (
                <div className='space-y-3'>
                  <Skeleton className='h-40 w-full' />
                  <Skeleton className='h-40 w-full' />
                </div>
              ) : (
                <div className='space-y-4'>
                  <div>
                    <div className='mb-1 text-sm font-medium'>
                      {t('Request body')}
                    </div>
                    <div className='bg-muted max-h-56 overflow-auto rounded-md border p-3'>
                      <pre className='text-xs break-all whitespace-pre-wrap'>
                        {detail.request_body || '-'}
                      </pre>
                    </div>
                  </div>
                  <div>
                    <div className='mb-1 text-sm font-medium'>
                      {t('Response body')}
                    </div>
                    <div className='bg-muted max-h-56 overflow-auto rounded-md border p-3'>
                      <pre className='text-xs break-all whitespace-pre-wrap'>
                        {detail.response_body || '-'}
                      </pre>
                    </div>
                  </div>
                </div>
              )}
            </DialogContent>
          </Dialog>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
