class Solution {
public:
    int carFleet(int target, vector<int>& position, vector<int>& speed) {
        vector<pair<int,double>> cars;
        for(int i=0;i<position.size();++i){
            cars.push_back({position[i],(double)(target-position[i])/speed[i]});
        }
        sort(cars.begin(),cars.end(),[&](const pair<int,double>& left, const pair<int,double>& right){
            return left.first<=right.first;
        });
        stack<double> st;
        for(int i=position.size()-1;i>=0;--i){
            double maxTime=cars[i].second;
            while(!st.empty() && maxTime<=st.top()){
                maxTime=max(maxTime,st.top());
                st.pop();
            }
            st.push(maxTime);
        }
        return st.size();
    }
};