export const delay = (ms: number) =>
  new Promise((resolve) => setTimeout(resolve, ms))

export const setRandomInterval = <TArgs extends any[]>(
  intervalFunction: (...args: TArgs) => void,
  setDelayCallback?: (delay: number) => void,
  minDelay: number = 0,
  maxDelay: number = 0,
  ...args: TArgs
) => {
  let timeout: NodeJS.Timeout

  const runInterval = () => {
    const timeoutFunction = () => {
      intervalFunction(...args)
      runInterval()
    }

    const delay =
      Math.floor(Math.random() * (maxDelay - minDelay + 1)) + minDelay
    if (setDelayCallback) setDelayCallback(delay)

    timeout = setTimeout(timeoutFunction, delay)
  }

  runInterval()

  return {
    clear() {
      clearTimeout(timeout)
    },
  }
}

export const T_MINS = 60000
