class Solution
{
public:
    int sumSubarrayMins(vector<int> &arr)
    {
        vector<int> left(arr.size(), -1), right(arr.size(), arr.size());
        stack<int> st;
        for (int i = 0; i < arr.size(); ++i)
        {
            while (!st.empty() && arr[st.top()] >= arr[i])
            {
                right[st.top()] = i;
                st.pop();
            }
            if (!st.empty())
                left[i] = st.top();
            st.push(i);
        }
        long ans = 0L;
        for (int i = 0; i < arr.size(); ++i)
        {
            ans += (long)arr[i] * (i - left[i]) * (right[i] - i);
        }
        return (int)(ans % (long)(1e9 + 7));
    }
};