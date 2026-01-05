import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Title,
  Tooltip,
} from 'chart.js'

export const initCharts = () => {
  ChartJS.register(Title,
    Tooltip,
    LineElement,
    CategoryScale,
    LinearScale,
    PointElement,
    Legend,
    Filler)
}
