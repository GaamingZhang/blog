class Solution
{
public:
    vector<int> nextGreaterElement(vector<int> &nums1, vector<int> &nums2)
    {
        vector<int> answer(nums1.size(), -1);
        unordered_map<int, int> greaterElement;
        stack<int> st;
        for (int i = 0; i < nums2.size(); ++i)
        {
            while (!st.empty() && nums2[i] > st.top())
            {
                greaterElement[st.top()] = nums2[i];
                st.pop();
            }
            st.push(nums2[i]);
            greaterElement[nums2[i]] = -1;
        }
        for (int i = 0; i < nums1.size(); ++i)
        {
            // if(greaterElement.find(nums1[i])!=greaterElement.end()){
            answer[i] = greaterElement[nums1[i]];
            //}
        }
        return answer;
    }
};