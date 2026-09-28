import type { Item } from "../../../bindings/github.com/yito0000/meal-cycle/internal/domain/shopping/models";

// Go 側で必要量 0 になるのは量を持たない材料の行だけ、という性質に依存する (add-shopping-pantry-section design.md D1)。
export function isPantryItem(item: Pick<Item, "amount">): boolean {
  return item.amount === 0;
}
