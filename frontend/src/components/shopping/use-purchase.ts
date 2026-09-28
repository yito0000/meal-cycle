import { useMutation, useQueryClient } from "@tanstack/react-query";
import { FridgeService } from "../../../bindings/github.com/yito0000/meal-cycle/internal/service";
import { type StockFormValues, toStockInput } from "../fridge/stock-form";
import { useInvalidateFridge } from "../fridge/use-fridge";

// 登録した食材の在庫の状態をその場で変えるため、S07 自身の一覧も捨てる (design.md D6)。
export function useRegisterPurchase() {
  const queryClient = useQueryClient();
  const invalidateFridge = useInvalidateFridge();
  return useMutation({
    mutationFn: (rows: StockFormValues[]) => FridgeService.AddMany(rows.map(toStockInput)),
    onSuccess: () =>
      Promise.all([invalidateFridge(), queryClient.invalidateQueries({ queryKey: ["shopping"] })]),
  });
}
