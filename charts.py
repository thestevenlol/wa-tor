import pandas as pd
import matplotlib.pyplot as plt

def read_tps_data(threads: int) -> pd.DataFrame:
    data = pd.read_csv(f"tpsMeasurementNThreads_{threads}.csv")
    return data

def plot_tps(thread_counts: list) -> plt:
    for thread_count in thread_counts:
        data = read_tps_data(thread_count)
        plt.plot(data['tps'], label=f'{thread_count} threads')
    plt.xlabel("Time (s)")
    plt.ylabel("TPS")
    plt.legend()
    plt.title("TPS vs Time")
    return plt

if __name__ == "__main__":
    threads = [1,2,4,8,16]
    plot_tps(threads).savefig(f"tps.png")