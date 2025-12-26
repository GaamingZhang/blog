class Solution {
public:
    int trap(vector<int>& height) {
        stack<int> st;
        int result=0;
        for(int i=0;i<height.size();++i){
            while(!st.empty() && height[i]>height[st.top()]){
                int bottom=height[st.top()];
                st.pop();
                if(st.empty()) break;
                result+=(min(height[st.top()],height[i])-bottom)*(i-st.top()-1);
            }
            st.push(i);
        }
        return result;
    }
};