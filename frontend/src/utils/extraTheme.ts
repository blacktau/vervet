
type ExtraTheme = {
  titleColor: string,
  ribbonColor: string,
  ribbonActiveColor: string,
  sidebarColor: string,
  splitColor: string,
}

export const extraLightTheme = {
  titleColor: '#F2F2F2',
  ribbonColor: '#F9F9F9',
  ribbonActiveColor: '#E3E3E3',
  sidebarColor: '#F2F2F2',
  splitColor: '#DADADA',
} as ExtraTheme

export const extraDarkTheme = {
  titleColor: '#262626',
  ribbonColor: '#2C2C2C',
  ribbonActiveColor: '#363636',
  sidebarColor: '#262626',
  splitColor: '#474747',
} as ExtraTheme

export const extraTheme = (dark: boolean) => {
  return dark ? extraDarkTheme : extraLightTheme
}
