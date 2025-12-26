class Solution
{
public:
    vector<int> searchRange(vector<int> &nums, int target)
    {
        int start = lowerBound(nums, target);
        if (start == nums.size() || nums[start] != target)
        {
            return {-1, -1};
        }
        int end = lowerBound(nums, target + 1) - 1;
        return {start, end};
    }
    int lowerBound(vector<int> &nums, int target)
    {
        int left = 0, right = nums.size();
        while (left < right)
        {
            int mid = left + (right - left) / 2;
            if (nums[mid] >= target)
            {
                right = mid;
            }
            else
            {
                left = mid + 1;
            }
        }
        return left;
    }
};