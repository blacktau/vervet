import { padStart } from 'lodash'
import { i18nGlobal } from '../i18n'

export const toHumanReadable = (duration: number) => {
  const days = Math.floor(duration / 86400)
  const hours = Math.floor((duration % 86400) / 3600)
  const minutes = Math.floor(((duration % 86400) % 3600) / 60)
  const seconds = Math.floor(duration % 60)
  const time = `${padStart(hours.toString(), 2, '0')}:${padStart(minutes.toString(), 2, '0')}:${padStart(seconds.toString(), 2, '0')}`
  if (days > 0) {
    return days + i18nGlobal.t('common.unit_day') + ' ' + time
  } else {
    return time
  }
}
