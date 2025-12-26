class Solution
{
public:
    int maximumCount(vector<int> &nums)
    {
        int zeroLowerBound = lowerBound(nums, 0);
        if (zeroLowerBound == nums.size())
            return nums.size();
        if (nums[zeroLowerBound] != 0)
        {
            return max(zeroLowerBound, (int)nums.size() - zeroLowerBound);
        }
        int zeroUpperBound = lowerBound(nums, 1) - 1;
        return max(zeroLowerBound, (int)nums.size() - zeroUpperBound - 1);
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