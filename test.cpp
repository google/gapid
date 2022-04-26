#include "minimal_state_tracker.h"
#include "state_block.h"
#include "transform.h"
#include "transform_test.h"

int main() {
  gapid2::transform<gapid2::transform_base> setup_null_transform(nullptr);
  gapid2::transform<gapid2::state_block> t2(&setup_null_transform);
  gapid2::transform<gapid2::minimal_state_tracker> t3(&setup_null_transform);
  gapid2::transform<gapid2::transform_test2> t4(&setup_null_transform);
}
