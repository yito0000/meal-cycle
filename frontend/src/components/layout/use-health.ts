import { useQuery } from "@tanstack/react-query";
import { HealthService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";

// 起動時の状態はプロセスの寿命中に変わらないので、取り直さない。
export function useHealth() {
  return useQuery({
    queryKey: ["health"],
    queryFn: () => HealthService.Ping(),
    staleTime: Number.POSITIVE_INFINITY,
  });
}
