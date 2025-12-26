class Solution
{
public:
    vector<int> maxNumber(vector<int> &nums1, vector<int> &nums2, int k)
    {
        int m = nums1.size(), n = nums2.size();
        vector<int> maxSubsequence(k, 0);
        int start = max(0, k - n), end = min(k, m);
        for (int i = start; i <= end; ++i)
        {
            vector<int> subsequence1(MaxSubsequence(nums1, i));
            vector<int> subsequence2(MaxSubsequence(nums2, k - i));
            vector<int> curMaxSubsequence(merge(subsequence1, subsequence2));
            if (compare(curMaxSubsequence, 0, maxSubsequence, 0) > 0)
            {
                maxSubsequence.swap(curMaxSubsequence);
            }
        }
        return maxSubsequence;
    }
    vector<int> MaxSubsequence(vector<int> &nums, int k)
    {
        vector<int> st;
        int remain = nums.size() - k;
        for (int i = 0; i < nums.size(); ++i)
        {
            while (!st.empty() && st.back() < nums[i] && remain > 0)
            {
                st.pop_back();
                --remain;
            }
            st.push_back(nums[i]);
        }
        return vector(st.begin(), st.begin() + k);
    }
    vector<int> merge(vector<int> &subsequence1, vector<int> &subsequence2)
    {
        int x = subsequence1.size(), y = subsequence2.size();
        if (x == 0)
            return subsequence2;
        if (y == 0)
            return subsequence1;
        int mergeLength = x + y;
        vector<int> merged(mergeLength);
        int index1 = 0, index2 = 0;
        for (int i = 0; i < mergeLength; ++i)
        {
            if (compare(subsequence1, index1, subsequence2, index2) > 0)
            {
                merged[i] = subsequence1[index1++];
            }
            else
            {
                merged[i] = subsequence2[index2++];
            }
        }
        return merged;
    }
    int compare(vector<int> &subsequence1, int index1, vector<int> &subsequence2, int index2)
    {
        int x = subsequence1.size(), y = subsequence2.size();
        while (index1 < x && index2 < y)
        {
            int difference = subsequence1[index1] - subsequence2[index2];
            if (difference != 0)
            {
                return difference;
            }
            ++index1;
            ++index2;
        }
        return (x - index1) - (y - index2);
    }
};