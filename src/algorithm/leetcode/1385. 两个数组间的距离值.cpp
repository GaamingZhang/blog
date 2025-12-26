class Solution
{
public:
    int findTheDistanceValue(vector<int> &arr1, vector<int> &arr2, int d)
    {
        int result = 0;
        sort(arr2.begin(), arr2.end(), less<int>());
        for (int i = 0; i < arr1.size(); ++i)
        {
            int leftIndex = lowerBound(arr2, arr1[i] - d);
            if (leftIndex == arr2.size() || arr2[leftIndex] > arr1[i] + d)
                ++result;
        }
        return result;
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